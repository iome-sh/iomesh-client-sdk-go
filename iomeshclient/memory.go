package iomeshclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// MemoryProductConfig is the v5 registry shape for tenant-scoped Palace products.
type MemoryProductConfig struct {
	ProductID         string `json:"product_id"`
	TenantID          string `json:"tenant_id"`
	PalaceRoot        string `json:"palace_root"`
	QdrantURL         string `json:"qdrant_url,omitempty"`
	QdrantCollection  string `json:"qdrant_collection,omitempty"`
	EmbeddingDim      int    `json:"embedding_dim,omitempty"`
	RecMemEnabled     bool   `json:"recmem_enabled,omitempty"`
	MaxWorkingEntries int    `json:"max_working_entries,omitempty"`
	CreatedAt         int64  `json:"created_at,omitempty"`
}

// MemoryEntityRef anchors a memory turn to an external entity (ticket, PR, account).
type MemoryEntityRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// MemoryEnvelope is the v5 memory stream / ingest payload shape.
// Temporal fields are optional (omitempty) for backward compatibility with older brokers/sidecars.
type MemoryEnvelope struct {
	Type          string  `json:"type"`
	SessionID     string  `json:"session_id,omitempty"`
	TurnID        string  `json:"turn_id,omitempty"`
	MemoryID      string  `json:"memory_id,omitempty"`
	Tier          int     `json:"tier,omitempty"`
	Role          string  `json:"role,omitempty"`
	Content       string  `json:"content,omitempty"`
	EmbeddingRef  string  `json:"embedding_ref,omitempty"`
	SurpriseScore float64 `json:"surprise_score,omitempty"`

	// Temporal envelope fields (optional; match platform memory envelope).
	EventTime      string            `json:"event_time,omitempty"`       // RFC3339 source-system fact time
	IngestedAt     string            `json:"ingested_at,omitempty"`      // RFC3339 palace capture time (usually server-set)
	SourceStream   string            `json:"source_stream,omitempty"`    // originating durable stream name
	SourceSeq      uint64            `json:"source_seq,omitempty"`       // originating broker seq
	SessionSeq     int               `json:"session_seq,omitempty"`      // monotonic order within session_id
	CausalParentID string            `json:"causal_parent_id,omitempty"` // prior memory/event id
	EntityRefs     []MemoryEntityRef `json:"entity_refs,omitempty"`      // ticket, PR, account anchors
	ValidFrom      string            `json:"valid_from,omitempty"`       // RFC3339 fact validity start
	ValidUntil     string            `json:"valid_until,omitempty"`      // RFC3339 fact validity end
}

// MemoryRetrieveRequest is the sync HTTP body for POST /v5/memory/retrieve.
// RequestMemoryRecall remains the async stream path (MEMORY_RPC publish).
type MemoryRetrieveRequest struct {
	TenantID   string `json:"tenant_id"`
	Query      string `json:"query"`
	Limit      int    `json:"limit,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	SessionSeq int    `json:"session_seq,omitempty"` // query session order for temporal recall
	Since      string `json:"since,omitempty"`       // RFC3339 inclusive lower bound
	Until      string `json:"until,omitempty"`       // RFC3339 inclusive upper bound
}

// MemoryHit is one recall result from the memory sidecar (POST /v5/memory/retrieve).
// Field names match the sidecar wire shape; Content is filled from Full/Summary when decoding helpers need a single string.
type MemoryHit struct {
	ID         string  `json:"id,omitempty"`
	MemoryID   string  `json:"memory_id,omitempty"` // alias some gateways may emit
	Summary    string  `json:"summary,omitempty"`
	Full       string  `json:"full,omitempty"`
	Content    string  `json:"content,omitempty"` // optional alternate content field
	Score      float64 `json:"score,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
	Timestamp  string  `json:"timestamp,omitempty"`
	TurnID     string  `json:"turn_id,omitempty"`
	EventTime  string  `json:"event_time,omitempty"`
	SessionSeq int     `json:"session_seq,omitempty"`
}

// MemoryRetrieveResponse is the sync retrieve JSON body.
type MemoryRetrieveResponse struct {
	Memories []MemoryHit `json:"memories"`
	// Path is the successful API path (/v1/memory/retrieve or /v5/memory/retrieve). Not JSON.
	Path string `json:"-"`
}

// MemoryRecallRequest is the async MEMORY_RPC publish body (RequestMemoryRecall / RequestMemoryRecallFull).
type MemoryRecallRequest struct {
	TenantID  string
	Query     string
	Limit     int
	SessionID string // optional temporal correlation (parity with iomesh-tui dogfood)
}

// MemoryIngestResponse is the sync ingest JSON body from POST /v5/memory/ingest.
type MemoryIngestResponse struct {
	Status   string `json:"status,omitempty"`
	MemoryID string `json:"memory_id,omitempty"`
	Tier     int    `json:"tier,omitempty"`
	Ingested int    `json:"ingested,omitempty"`
}

const (
	memoryEnvelopeIngest = "memory_ingest"
	memoryEnvelopeRecall = "memory_recall"
	streamMemoryIngest   = "MEMORY_INGEST"
	streamMemoryRPC      = "MEMORY_RPC"

	// Public alias then sidecar-stable path (parity with iomesh-tui RetrieveMemory).
	pathMemoryRetrieveV1 = "/v1/memory/retrieve"
	pathMemoryRetrieveV5 = "/v5/memory/retrieve"
	pathMemoryIngestV1   = "/v1/memory/ingest"
	pathMemoryIngestV5   = "/v5/memory/ingest"
)

// RegisterMemoryProduct registers a memory DataProduct via POST /v5/registry/memory-products.
// A 409 conflict is treated as success (idempotent re-register).
func (c *Client) RegisterMemoryProduct(ctx context.Context, cfg MemoryProductConfig) error {
	if cfg.ProductID == "" {
		return errors.New("iomeshclient: product_id required")
	}
	if cfg.TenantID == "" {
		return errors.New("iomeshclient: tenant_id required")
	}
	if cfg.PalaceRoot == "" {
		return errors.New("iomeshclient: palace_root required")
	}

	var resp MemoryProductConfig
	if err := c.doJSON(ctx, http.MethodPost, "/v5/registry/memory-products", cfg, &resp); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict {
			return nil
		}
		return err
	}
	return nil
}

// ListMemoryProducts returns memory products for tenantID via GET /v5/registry/memory-products.
func (c *Client) ListMemoryProducts(ctx context.Context, tenantID string) ([]MemoryProductConfig, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errors.New("iomeshclient: tenant_id required")
	}

	path := fmt.Sprintf("/v5/registry/memory-products?tenant_id=%s", url.QueryEscape(tenantID))
	var products []MemoryProductConfig
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &products); err != nil {
		return nil, err
	}
	if products == nil {
		products = []MemoryProductConfig{}
	}
	return products, nil
}

// PublishMemoryIngest publishes a memory_ingest envelope to MEMORY_INGEST.
func (c *Client) PublishMemoryIngest(ctx context.Context, tenantID string, env MemoryEnvelope) (*PubAck, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errors.New("iomeshclient: tenant_id required")
	}
	if strings.TrimSpace(env.Type) == "" {
		env.Type = memoryEnvelopeIngest
	}
	if strings.TrimSpace(env.Content) == "" {
		return nil, errors.New("iomeshclient: content required for memory ingest")
	}

	payload, err := json.Marshal(env)
	if err != nil {
		return nil, err
	}
	subject := tenantID + ".memory.ingest.turn"
	return c.Publish(ctx, streamMemoryIngest, subject, payload)
}

// DualWriteMemoryOptions controls DualWriteMemoryTurn.
type DualWriteMemoryOptions struct {
	// Sync also performs IngestMemoryTurn against SyncClient (fail-open).
	// When false, only async MEMORY_INGEST publish runs.
	Sync bool
	// SyncClient is the client used for sync ingest (memory sidecar URL).
	// When nil and Sync is true, the receiver client is used (same endpoint as mesh).
	SyncClient *Client
}

// DualWriteMemoryResult is the outcome of DualWriteMemoryTurn.
// Async failure is returned as the function error; sync failures are fail-open in SyncErr.
type DualWriteMemoryResult struct {
	Async   *PubAck
	Sync    *MemoryIngestResponse
	SyncErr error
}

// DualWriteMemoryTurn publishes async MEMORY_INGEST (required path) and optionally
// sync POST /v1|/v5/memory/ingest (Palace write) with fail-open semantics matching
// iomesh-tui agent dual_write: stream durable first; sidecar best-effort.
func (c *Client) DualWriteMemoryTurn(ctx context.Context, tenantID string, env MemoryEnvelope, opts DualWriteMemoryOptions) (*DualWriteMemoryResult, error) {
	ack, err := c.PublishMemoryIngest(ctx, tenantID, env)
	if err != nil {
		return nil, err
	}
	out := &DualWriteMemoryResult{Async: ack}
	if !opts.Sync {
		return out, nil
	}
	syncC := opts.SyncClient
	if syncC == nil {
		syncC = c
	}
	resp, syncErr := syncC.IngestMemoryTurn(ctx, tenantID, env)
	out.Sync = resp
	out.SyncErr = syncErr
	return out, nil
}

// RequestMemoryRecall publishes an async memory_recall request to MEMORY_RPC.
// For session correlation, use RequestMemoryRecallFull. For sync hits, use RetrieveMemory.
func (c *Client) RequestMemoryRecall(ctx context.Context, tenantID, query string, limit int) (*PubAck, error) {
	return c.RequestMemoryRecallFull(ctx, MemoryRecallRequest{
		TenantID: tenantID,
		Query:    query,
		Limit:    limit,
	})
}

// RequestMemoryRecallFull publishes async MEMORY_RPC with optional session_id (TUI dogfood parity).
func (c *Client) RequestMemoryRecallFull(ctx context.Context, req MemoryRecallRequest) (*PubAck, error) {
	tenantID := strings.TrimSpace(req.TenantID)
	if tenantID == "" {
		return nil, errors.New("iomeshclient: tenant_id required")
	}
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return nil, errors.New("iomeshclient: query required")
	}

	body := map[string]any{
		"type":      memoryEnvelopeRecall,
		"tenant_id": tenantID,
		"query":     query,
	}
	if req.Limit > 0 {
		body["limit"] = req.Limit
	}
	if sid := strings.TrimSpace(req.SessionID); sid != "" {
		body["session_id"] = sid
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	subject := tenantID + ".memory.retrieve.request"
	return c.Publish(ctx, streamMemoryRPC, subject, payload)
}

// RetrieveMemory performs synchronous hybrid recall against the memory sidecar HTTP API.
// Tries POST /v1/memory/retrieve then /v5/memory/retrieve (iomesh-tui / platform parity).
// Prefer this over RequestMemoryRecall when the caller needs hits in-process.
// Query may be empty when SessionID is set (session-scoped temporal slice).
func (c *Client) RetrieveMemory(ctx context.Context, req MemoryRetrieveRequest) (*MemoryRetrieveResponse, error) {
	req.TenantID = strings.TrimSpace(req.TenantID)
	req.Query = strings.TrimSpace(req.Query)
	req.SessionID = strings.TrimSpace(req.SessionID)
	if req.TenantID == "" {
		return nil, errors.New("iomeshclient: tenant_id required")
	}
	if req.Query == "" && req.SessionID == "" {
		return nil, errors.New("iomeshclient: query or session_id required")
	}

	// Wire type for recall so sidecars that validate envelope type accept the request.
	body := map[string]any{
		"tenant_id": req.TenantID,
		"type":      memoryEnvelopeRecall,
		"query":     req.Query,
	}
	if req.Limit > 0 {
		body["limit"] = req.Limit
	}
	if req.SessionID != "" {
		body["session_id"] = req.SessionID
	}
	if req.SessionSeq != 0 {
		body["session_seq"] = req.SessionSeq
	}
	if since := strings.TrimSpace(req.Since); since != "" {
		body["since"] = since
	}
	if until := strings.TrimSpace(req.Until); until != "" {
		body["until"] = until
	}

	var lastErr error
	for _, path := range []string{pathMemoryRetrieveV1, pathMemoryRetrieveV5} {
		var resp MemoryRetrieveResponse
		err := c.doJSON(ctx, http.MethodPost, path, body, &resp)
		if err == nil {
			if resp.Memories == nil {
				resp.Memories = []MemoryHit{}
			}
			resp.Path = path
			return &resp, nil
		}
		lastErr = err
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			continue // try next path
		}
		// Non-404: still try v5 once if v1 failed for other reasons? Prefer fail fast on 4xx != 404.
		if errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 && apiErr.StatusCode != http.StatusNotFound {
			return nil, err
		}
		// transport / 5xx: try next path once
		continue
	}
	if lastErr == nil {
		lastErr = errors.New("iomeshclient: memory retrieve: no path succeeded")
	}
	return nil, lastErr
}

// IngestMemoryTurn performs a synchronous single-turn ingest via POST /v1 then /v5 memory/ingest.
// PublishMemoryIngest remains the async stream path (MEMORY_INGEST publish).
// Temporal fields on env (event_time, session_seq, …) are forwarded when set.
func (c *Client) IngestMemoryTurn(ctx context.Context, tenantID string, env MemoryEnvelope) (*MemoryIngestResponse, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errors.New("iomeshclient: tenant_id required")
	}
	if strings.TrimSpace(env.Content) == "" {
		return nil, errors.New("iomeshclient: content required for memory ingest")
	}
	if strings.TrimSpace(env.Type) == "" {
		env.Type = memoryEnvelopeIngest
	}

	// Sidecar accepts tenant_id + envelope fields at the top level (and optional turns[]).
	body := map[string]any{
		"tenant_id": tenantID,
		"type":      env.Type,
	}
	if env.SessionID != "" {
		body["session_id"] = env.SessionID
	}
	if env.TurnID != "" {
		body["turn_id"] = env.TurnID
	}
	if env.MemoryID != "" {
		body["memory_id"] = env.MemoryID
	}
	if env.Role != "" {
		body["role"] = env.Role
	}
	if env.Content != "" {
		body["content"] = env.Content
	}
	if env.Tier != 0 {
		body["tier"] = env.Tier
	}
	if env.EventTime != "" {
		body["event_time"] = env.EventTime
	}
	if env.IngestedAt != "" {
		body["ingested_at"] = env.IngestedAt
	}
	if env.SourceStream != "" {
		body["source_stream"] = env.SourceStream
	}
	if env.SourceSeq != 0 {
		body["source_seq"] = env.SourceSeq
	}
	if env.SessionSeq != 0 {
		body["session_seq"] = env.SessionSeq
	}
	if env.CausalParentID != "" {
		body["causal_parent_id"] = env.CausalParentID
	}
	if len(env.EntityRefs) > 0 {
		body["entity_refs"] = env.EntityRefs
	}
	if env.ValidFrom != "" {
		body["valid_from"] = env.ValidFrom
	}
	if env.ValidUntil != "" {
		body["valid_until"] = env.ValidUntil
	}

	var lastErr error
	for _, path := range []string{pathMemoryIngestV1, pathMemoryIngestV5} {
		var resp MemoryIngestResponse
		err := c.doJSON(ctx, http.MethodPost, path, body, &resp)
		if err == nil {
			return &resp, nil
		}
		lastErr = err
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			continue
		}
		if errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 && apiErr.StatusCode != http.StatusNotFound {
			return nil, err
		}
		continue
	}
	if lastErr == nil {
		lastErr = errors.New("iomeshclient: memory ingest: no path succeeded")
	}
	return nil, lastErr
}
