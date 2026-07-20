package iomeshclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
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

// WaitReadyOptions configures WaitReady / WaitReadyElapsed polling.
type WaitReadyOptions struct {
	// Interval between probe attempts. Default 500ms when zero or negative.
	Interval time.Duration
	// RequireHealth also requires Health() to succeed each attempt (after Ready).
	RequireHealth bool
}

// WaitReady polls Ready until it succeeds or ctx is done.
// When RequireHealth is set, Health must also succeed on the same attempt.
// Returns the last probe error wrapped with ctx.Err() when the deadline expires,
// or ctx.Err() if cancelled with no prior probe error.
// Implemented via WaitReadyElapsed (elapsed discarded).
func (c *Client) WaitReady(ctx context.Context, opts WaitReadyOptions) error {
	_, err := c.WaitReadyElapsed(ctx, opts)
	return err
}

// WaitReadyElapsed is like WaitReady but also returns how long waiting took
// (until success or error). On success, elapsed is wall time until the first
// successful probe. On error or cancel, elapsed is time until failure.
// Nil client returns (0, error). Elapsed is always >= 0.
func (c *Client) WaitReadyElapsed(ctx context.Context, opts WaitReadyOptions) (elapsed time.Duration, err error) {
	start := time.Now()
	if c == nil {
		return 0, fmt.Errorf("iomeshclient: nil client")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	interval := opts.Interval
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}

	var last error
	for {
		if err := ctx.Err(); err != nil {
			elapsed = time.Since(start)
			if last != nil {
				return elapsed, fmt.Errorf("iomeshclient: wait ready: %w (last: %v)", err, last)
			}
			return elapsed, fmt.Errorf("iomeshclient: wait ready: %w", err)
		}

		err := c.Ready(ctx)
		if err == nil && opts.RequireHealth {
			err = c.Health(ctx)
		}
		if err == nil {
			return time.Since(start), nil
		}
		last = err

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			elapsed = time.Since(start)
			if last != nil {
				return elapsed, fmt.Errorf("iomeshclient: wait ready: %w (last: %v)", ctx.Err(), last)
			}
			return elapsed, fmt.Errorf("iomeshclient: wait ready: %w", ctx.Err())
		case <-timer.C:
		}
	}
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
