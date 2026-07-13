package aionclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Processor type constants for v3 registry processors.
const (
	ProcessorTypeFilter = "filter"
	ProcessorTypeMap    = "map"
	ProcessorTypeEnrich = "enrich"
)

// ProcessorConfig defines an in-stream enrichment processor bound to a stream pair.
type ProcessorConfig struct {
	ID           string `json:"id"`
	SourceStream string `json:"source_stream"`
	TargetStream string `json:"target_stream,omitempty"`
	Tenant       string `json:"tenant"`
	Type         string `json:"type"`
	ConfigJSON   string `json:"config_json,omitempty"`
}

// DataProduct is registry metadata for a governed stream or subject family.
type DataProduct struct {
	ID                 string    `json:"id"`
	TenantID           string    `json:"tenant_id"`
	Name               string    `json:"name"`
	Domain             string    `json:"domain"`
	Owner              string    `json:"owner"`
	SchemaRef          string    `json:"schema_ref,omitempty"`
	UpstreamProductIDs []string  `json:"upstream_product_ids,omitempty"`
	Subjects           []string  `json:"subjects"`
	StreamName         string    `json:"stream_name,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

// LiveView extends DataProduct with enrichment lineage and warm-tier metadata.
type LiveView struct {
	DataProduct
	ProcessorIDs     []string `json:"processor_ids,omitempty"`
	FreshnessSLOSec  int      `json:"freshness_slo_sec,omitempty"`
	MaterializedPath string   `json:"materialized_path,omitempty"`
}

// RegisterProcessor registers a processor via POST /v3/registry/processors.
// A 409 conflict is treated as success (idempotent re-register).
func (c *Client) RegisterProcessor(ctx context.Context, cfg ProcessorConfig) error {
	if cfg.ID == "" {
		return errors.New("aionclient: processor id required")
	}
	if cfg.SourceStream == "" {
		return errors.New("aionclient: source_stream required")
	}
	if cfg.Tenant == "" {
		return errors.New("aionclient: tenant required")
	}
	if cfg.Type == "" {
		return errors.New("aionclient: type required")
	}

	var resp ProcessorConfig
	if err := c.doJSON(ctx, http.MethodPost, "/v3/registry/processors", cfg, &resp); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict {
			return nil
		}
		return err
	}
	return nil
}

// ListLiveViews returns live views for tenantID via GET /v3/registry/liveviews.
func (c *Client) ListLiveViews(ctx context.Context, tenantID string) ([]LiveView, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errors.New("aionclient: tenant_id required")
	}

	path := fmt.Sprintf("/v3/registry/liveviews?tenant_id=%s", url.QueryEscape(tenantID))
	var views []LiveView
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &views); err != nil {
		return nil, err
	}
	if views == nil {
		views = []LiveView{}
	}
	return views, nil
}