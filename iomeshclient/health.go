package iomeshclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Health checks GET /health on the broker base URL.
// Returns nil when the broker is reachable and responds 200.
func (c *Client) Health(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("iomeshclient: nil client")
	}
	return c.getStatus(ctx, "/health")
}

// Ready checks GET /ready then /readyz (parity with iomesh-tui dogfood).
// A 404 on both paths is an error; other non-OK statuses fail immediately.
func (c *Client) Ready(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("iomeshclient: nil client")
	}
	var last404 error
	for _, path := range []string{"/ready", "/readyz"} {
		err := c.getStatus(ctx, path)
		if err == nil {
			return nil
		}
		if strings.Contains(err.Error(), "http 404") {
			last404 = err
			continue
		}
		return err
	}
	if last404 != nil {
		return last404
	}
	return fmt.Errorf("iomeshclient: ready: http 404")
}

func (c *Client) getStatus(ctx context.Context, path string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	c.applyAuthHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	if resp.StatusCode == http.StatusOK {
		return nil
	}
	name := strings.TrimPrefix(path, "/")
	return fmt.Errorf("iomeshclient: %s: http %d", name, resp.StatusCode)
}
