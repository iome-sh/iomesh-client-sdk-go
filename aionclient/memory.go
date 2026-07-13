package aionclient

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

// MemoryEnvelope is the v5 memory stream / ingest payload shape.
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
}

const (
	memoryEnvelopeIngest = "memory_ingest"
	memoryEnvelopeRecall = "memory_recall"
	streamMemoryIngest     = "MEMORY_INGEST"
	streamMemoryRPC        = "MEMORY_RPC"
)

// RegisterMemoryProduct registers a memory DataProduct via POST /v5/registry/memory-products.
// A 409 conflict is treated as success (idempotent re-register).
func (c *Client) RegisterMemoryProduct(ctx context.Context, cfg MemoryProductConfig) error {
	if cfg.ProductID == "" {
		return errors.New("aionclient: product_id required")
	}
	if cfg.TenantID == "" {
		return errors.New("aionclient: tenant_id required")
	}
	if cfg.PalaceRoot == "" {
		return errors.New("aionclient: palace_root required")
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
		return nil, errors.New("aionclient: tenant_id required")
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
		return nil, errors.New("aionclient: tenant_id required")
	}
	if strings.TrimSpace(env.Type) == "" {
		env.Type = memoryEnvelopeIngest
	}
	if strings.TrimSpace(env.Content) == "" {
		return nil, errors.New("aionclient: content required for memory ingest")
	}

	payload, err := json.Marshal(env)
	if err != nil {
		return nil, err
	}
	subject := tenantID + ".memory.ingest.turn"
	return c.Publish(ctx, streamMemoryIngest, subject, payload)
}

// RequestMemoryRecall publishes an async memory_recall request to MEMORY_RPC.
func (c *Client) RequestMemoryRecall(ctx context.Context, tenantID, query string, limit int) (*PubAck, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errors.New("aionclient: tenant_id required")
	}
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("aionclient: query required")
	}

	body := map[string]any{
		"type":      memoryEnvelopeRecall,
		"tenant_id": tenantID,
		"query":     query,
	}
	if limit > 0 {
		body["limit"] = limit
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	subject := tenantID + ".memory.retrieve.request"
	return c.Publish(ctx, streamMemoryRPC, subject, payload)
}