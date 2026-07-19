package iomeshclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// RetentionLimits caps streams by max message count and/or age (default).
	RetentionLimits = "limits"
	// RetentionWorkQueue removes messages from redelivery once any consumer acks.
	RetentionWorkQueue = "workqueue"
)

// StreamConfig describes a stream to create via the HTTP API.
type StreamConfig struct {
	Name        string   `json:"name"`
	Subjects    []string `json:"subjects"`
	Retention   string   `json:"retention,omitempty"`
	Partitions  int      `json:"partitions,omitempty"`
	MaxMsgs     *int64   `json:"max_msgs,omitempty"`
	MaxAgeSec   *int64   `json:"max_age_sec,omitempty"`
	Description string   `json:"description,omitempty"`
}

// StreamInfo is broker stream metadata from GET /v1/streams and GET /v1/streams/{name}.
// Wire shape matches the broker stream response (name, subjects, stats, retention knobs).
type StreamInfo struct {
	Name        string    `json:"name"`
	Subjects    []string  `json:"subjects"`
	Retention   string    `json:"retention,omitempty"`
	Partitions  int       `json:"partitions,omitempty"`
	MaxMsgs     *int64    `json:"max_msgs,omitempty"`
	MaxAgeSec   *int64    `json:"max_age_sec,omitempty"`
	Description string    `json:"description,omitempty"`
	Messages    uint64    `json:"messages"`
	FirstSeq    uint64    `json:"first_seq"`
	LastSeq     uint64    `json:"last_seq"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateStream registers a stream via POST /v1/streams.
// On 201, decodes StreamInfo from the response body (broker stream response).
// On 409 conflict, treats as success and best-effort GETs the stream
// (may return nil info if get fails).
func (c *Client) CreateStream(ctx context.Context, cfg StreamConfig) (*StreamInfo, error) {
	if c == nil {
		return nil, errors.New("iomeshclient: nil client")
	}
	if cfg.Name == "" {
		return nil, errors.New("iomeshclient: stream name required")
	}
	if len(cfg.Subjects) == 0 {
		return nil, errors.New("iomeshclient: subjects required")
	}

	var info StreamInfo
	err := c.doJSON(ctx, http.MethodPost, "/v1/streams", cfg, &info)
	if err == nil {
		if info.Name == "" {
			info.Name = cfg.Name // defensive when broker omits name
		}
		return &info, nil
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict {
		got, gerr := c.GetStream(ctx, cfg.Name)
		if gerr == nil {
			return got, nil
		}
		return nil, nil // conflict success without metadata
	}
	return nil, err
}

// EnsureStream creates the stream if it does not already exist.
// Same semantics as CreateStream (including 409 → success, optional StreamInfo).
func (c *Client) EnsureStream(ctx context.Context, cfg StreamConfig) (*StreamInfo, error) {
	return c.CreateStream(ctx, cfg)
}

// ListStreams returns all streams via GET /v1/streams.
// Unlike fail-open helpers (catalog/context/policy), this is explicit discovery:
// non-2xx returns *APIError and callers must handle errors (not an empty list).
// Accepts a JSON array body, or optionally an envelope {"streams":[...]}.
func (c *Client) ListStreams(ctx context.Context) ([]StreamInfo, error) {
	if c == nil {
		return nil, errors.New("iomeshclient: nil client")
	}

	var raw json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "/v1/streams", nil, &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return []StreamInfo{}, nil
	}

	var streams []StreamInfo
	if err := json.Unmarshal(raw, &streams); err == nil {
		if streams == nil {
			streams = []StreamInfo{}
		}
		return streams, nil
	}

	var env struct {
		Streams []StreamInfo `json:"streams"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Streams == nil {
		env.Streams = []StreamInfo{}
	}
	return env.Streams, nil
}

// GetStream returns one stream via GET /v1/streams/{name}.
// Empty name and nil client return errors. Non-2xx (including 404) returns *APIError.
func (c *Client) GetStream(ctx context.Context, name string) (*StreamInfo, error) {
	if c == nil {
		return nil, errors.New("iomeshclient: nil client")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("iomeshclient: stream name required")
	}

	path := "/v1/streams/" + url.PathEscape(name)
	var info StreamInfo
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// DeleteStream removes a stream via DELETE /v1/streams/{name}.
// Empty name / nil client → error. Non-2xx → *APIError (404 if missing).
// 204 No Content is success (doJSON handles empty body on 2xx).
func (c *Client) DeleteStream(ctx context.Context, name string) error {
	if c == nil {
		return errors.New("iomeshclient: nil client")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("iomeshclient: stream name required")
	}

	path := "/v1/streams/" + url.PathEscape(name)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil)
}

// StreamMessage is one message from stream replay/list (GET /v1/streams/{name}/messages).
// Payload is decoded from the wire base64 string; invalid base64 falls back to raw string bytes.
type StreamMessage struct {
	Stream    string
	Seq       uint64
	Subject   string
	Partition int
	Payload   []byte
	Headers   map[string]string
	Timestamp time.Time
}

// ListStreamMessagesOptions configures stream message replay range.
// Zero values map to broker-friendly defaults: FromSeq 0→1, ToSeq 0→last,
// Limit 0→100 (capped at 1000 client-side).
type ListStreamMessagesOptions struct {
	FromSeq uint64 // 0 → 1
	ToSeq   uint64 // 0 → broker last
	Limit   int    // 0 → 100; max 1000
}

// listStreamMessagesWire is the broker JSON envelope for message list/replay.
type listStreamMessagesWire struct {
	Messages []streamMessageWire `json:"messages"`
}

type streamMessageWire struct {
	Stream    string            `json:"stream"`
	Seq       uint64            `json:"seq"`
	Subject   string            `json:"subject"`
	Partition int               `json:"partition"`
	Payload   string            `json:"payload"`
	Headers   map[string]string `json:"headers"`
	Timestamp time.Time         `json:"timestamp"`
}

// ListStreamMessages returns messages from a stream via GET /v1/streams/{name}/messages
// (broker stream replay / read-range). Query: from_seq (default 1), to_seq (0=last),
// limit (default 100, max 1000). Nil client / empty stream → error. Non-2xx → *APIError.
// applyAuthHeaders (tenant etc.) run via doJSON — some brokers gate replay on tenant.
func (c *Client) ListStreamMessages(ctx context.Context, stream string, opts ListStreamMessagesOptions) ([]StreamMessage, error) {
	if c == nil {
		return nil, errors.New("iomeshclient: nil client")
	}
	stream = strings.TrimSpace(stream)
	if stream == "" {
		return nil, errors.New("iomeshclient: stream name required")
	}

	fromSeq := opts.FromSeq
	if fromSeq == 0 {
		fromSeq = 1
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	q := url.Values{}
	q.Set("from_seq", strconv.FormatUint(fromSeq, 10))
	q.Set("to_seq", strconv.FormatUint(opts.ToSeq, 10))
	q.Set("limit", strconv.Itoa(limit))

	path := "/v1/streams/" + url.PathEscape(stream) + "/messages?" + q.Encode()
	var wire listStreamMessagesWire
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &wire); err != nil {
		return nil, err
	}
	if wire.Messages == nil {
		return []StreamMessage{}, nil
	}

	out := make([]StreamMessage, len(wire.Messages))
	for i, m := range wire.Messages {
		out[i] = StreamMessage{
			Stream:    m.Stream,
			Seq:       m.Seq,
			Subject:   m.Subject,
			Partition: m.Partition,
			Payload:   decodeStreamPayload(m.Payload),
			Headers:   m.Headers,
			Timestamp: m.Timestamp,
		}
	}
	return out, nil
}

// decodeStreamPayload decodes a base64 payload string; on invalid base64 returns raw string bytes.
func decodeStreamPayload(s string) []byte {
	if s == "" {
		return []byte{}
	}
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b
	}
	return []byte(s)
}

// Pub publishes an ephemeral fire-and-forget message via POST /v1/pub.
func (c *Client) Pub(ctx context.Context, subject string, payload []byte, headers map[string]string) error {
	if subject == "" {
		return errors.New("iomeshclient: subject required")
	}

	req := ephemeralPubRequest{
		Subject: subject,
		Payload: string(payload),
		Headers: headers,
	}
	return c.doJSON(ctx, http.MethodPost, "/v1/pub", req, new(struct{}))
}

type ephemeralPubRequest struct {
	Subject string            `json:"subject"`
	Payload string            `json:"payload"`
	Headers map[string]string `json:"headers,omitempty"`
}
