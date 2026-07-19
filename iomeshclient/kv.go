package iomeshclient

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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

// BucketInfo is broker KV bucket metadata from create responses.
type BucketInfo struct {
	Name       string `json:"name"`
	MaxBytes   *int64 `json:"max_bytes,omitempty"`
	History    int    `json:"history,omitempty"`
	TTLSeconds *int64 `json:"ttl_seconds,omitempty"`
}

// CreateBucketConfig optionally tunes a new KV bucket.
type CreateBucketConfig struct {
	MaxBytes   *int64 `json:"max_bytes,omitempty"`
	History    int    `json:"history,omitempty"`
	TTLSeconds *int64 `json:"ttl_seconds,omitempty"`
}

// CreateBucket registers a KV bucket via POST /v1/kv/{name}.
// On 201, decodes BucketInfo from the response body.
// On 409 conflict, treats as success and returns &BucketInfo{Name: name} (name only).
// Empty name / nil client → error.
//
// Pre-1.0 signature change: previously returned only error. Callers that assigned a
// single return value must update to (*BucketInfo, error).
func (c *Client) CreateBucket(ctx context.Context, name string, cfg ...CreateBucketConfig) (*BucketInfo, error) {
	if c == nil {
		return nil, errors.New("iomeshclient: nil client")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("iomeshclient: bucket name required")
	}

	var req *createBucketRequest
	if len(cfg) > 0 {
		req = &createBucketRequest{
			MaxBytes:   cfg[0].MaxBytes,
			History:    cfg[0].History,
			TTLSeconds: cfg[0].TTLSeconds,
		}
	}

	path := "/v1/kv/" + url.PathEscape(name)
	var info BucketInfo
	err := c.doJSON(ctx, http.MethodPost, path, req, &info)
	if err == nil {
		if info.Name == "" {
			info.Name = name // defensive when broker omits name
		}
		return &info, nil
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict {
		return &BucketInfo{Name: name}, nil
	}
	return nil, err
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
	path := "/v1/kv/" + url.PathEscape(bucket)
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
