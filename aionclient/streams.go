package aionclient

import (
	"context"
	"errors"
	"net/http"
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

// CreateStream registers a stream via POST /v1/streams. A 409 conflict is treated as success.
func (c *Client) CreateStream(ctx context.Context, cfg StreamConfig) error {
	if cfg.Name == "" {
		return errors.New("aionclient: stream name required")
	}
	if len(cfg.Subjects) == 0 {
		return errors.New("aionclient: subjects required")
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

// Pub publishes an ephemeral fire-and-forget message via POST /v1/pub.
func (c *Client) Pub(ctx context.Context, subject string, payload []byte, headers map[string]string) error {
	if subject == "" {
		return errors.New("aionclient: subject required")
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
