package iomeshclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
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
// Wire shape matches aion streamResponse (name, subjects, stats, retention knobs).
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

// CreateStream registers a stream via POST /v1/streams. A 409 conflict is treated as success.
func (c *Client) CreateStream(ctx context.Context, cfg StreamConfig) error {
	if c == nil {
		return errors.New("iomeshclient: nil client")
	}
	if cfg.Name == "" {
		return errors.New("iomeshclient: stream name required")
	}
	if len(cfg.Subjects) == 0 {
		return errors.New("iomeshclient: subjects required")
	}

	var resp struct{}
	if err := c.doJSON(ctx, http.MethodPost, "/v1/streams", cfg, &resp); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict {
			return nil
		}
		return err
	}
	return nil
}

// EnsureStream creates the stream if it does not already exist.
func (c *Client) EnsureStream(ctx context.Context, cfg StreamConfig) error {
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
