package aionclient

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// KVEntry is a versioned key-value record from the broker.
type KVEntry struct {
	Bucket    string    `json:"bucket"`
	Key       string    `json:"key"`
	Value     []byte    `json:"value"`
	Revision  uint64    `json:"revision"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateBucketConfig optionally tunes a new KV bucket.
type CreateBucketConfig struct {
	MaxBytes   *int64 `json:"max_bytes,omitempty"`
	History    int    `json:"history,omitempty"`
	TTLSeconds *int64 `json:"ttl_seconds,omitempty"`
}

// CreateBucket registers a KV bucket. Duplicate bucket creation is treated as success.
func (c *Client) CreateBucket(ctx context.Context, name string, cfg ...CreateBucketConfig) error {
	var req *createBucketRequest
	if len(cfg) > 0 {
		req = &createBucketRequest{
			MaxBytes:   cfg[0].MaxBytes,
			History:    cfg[0].History,
			TTLSeconds: cfg[0].TTLSeconds,
		}
	}

	path := "/v1/kv/" + name
	var resp bucketResponse
	if err := c.doJSON(ctx, http.MethodPost, path, req, &resp); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict {
			return nil
		}
		return err
	}
	return nil
}

func kvKeyPath(bucket, key string) string {
	return fmt.Sprintf("/v1/kv/%s/%s", url.PathEscape(bucket), url.PathEscape(key))
}

// Put writes a key in bucket using JSON base64 encoding expected by the HTTP API.
func (c *Client) Put(ctx context.Context, bucket, key string, value []byte) (uint64, error) {
	body := putKVRequest{Value: base64.StdEncoding.EncodeToString(value)}
	path := kvKeyPath(bucket, key)

	var resp putKVResponse
	if err := c.doJSON(ctx, http.MethodPut, path, body, &resp); err != nil {
		return 0, err
	}
	return resp.Revision, nil
}

// Get returns the current value for key in bucket.
func (c *Client) Get(ctx context.Context, bucket, key string) (*KVEntry, error) {
	path := kvKeyPath(bucket, key)

	var entry KVEntry
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// Delete removes key from bucket.
func (c *Client) Delete(ctx context.Context, bucket, key string) error {
	path := kvKeyPath(bucket, key)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil)
}

// ListKeys returns keys in bucket, optionally filtered by prefix.
func (c *Client) ListKeys(ctx context.Context, bucket, prefix string) ([]string, error) {
	path := "/v1/kv/" + bucket
	if prefix != "" {
		path += "?prefix=" + url.QueryEscape(prefix)
	}

	var resp listKeysResponse
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Keys, nil
}

type createBucketRequest struct {
	MaxBytes   *int64 `json:"max_bytes,omitempty"`
	History    int    `json:"history,omitempty"`
	TTLSeconds *int64 `json:"ttl_seconds,omitempty"`
}

type bucketResponse struct {
	Name       string `json:"name"`
	MaxBytes   *int64 `json:"max_bytes,omitempty"`
	History    int    `json:"history,omitempty"`
	TTLSeconds *int64 `json:"ttl_seconds,omitempty"`
}

type putKVRequest struct {
	Value string `json:"value"`
}

type putKVResponse struct {
	Bucket   string `json:"bucket"`
	Key      string `json:"key"`
	Revision uint64 `json:"revision"`
}

type listKeysResponse struct {
	Keys []string `json:"keys"`
}
