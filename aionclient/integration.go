package aionclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// IcebergCatalogRef is the v4 registry shape for a warm-tier Iceberg table.
type IcebergCatalogRef struct {
	ID           string `json:"id"`
	TenantID     string `json:"tenant_id"`
	Namespace    string `json:"namespace"`
	TableName    string `json:"table_name"`
	S3Prefix     string `json:"s3_prefix"`
	SchemaJSON   string `json:"schema_json"`
	RegisteredAt int64  `json:"registered_at,omitempty"`
}

// RegisterIcebergRef registers a catalog ref via POST /v4/registry/iceberg.
// A 409 conflict is treated as success (idempotent re-register).
func (c *Client) RegisterIcebergRef(ctx context.Context, ref IcebergCatalogRef) error {
	if ref.ID == "" {
		return errors.New("aionclient: iceberg ref id required")
	}
	if ref.TenantID == "" {
		return errors.New("aionclient: tenant_id required")
	}
	if ref.Namespace == "" {
		return errors.New("aionclient: namespace required")
	}
	if ref.TableName == "" {
		return errors.New("aionclient: table_name required")
	}
	if ref.S3Prefix == "" {
		return errors.New("aionclient: s3_prefix required")
	}
	if ref.SchemaJSON == "" {
		return errors.New("aionclient: schema_json required")
	}

	var resp IcebergCatalogRef
	if err := c.doJSON(ctx, http.MethodPost, "/v4/registry/iceberg", ref, &resp); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict {
			return nil
		}
		return err
	}
	return nil
}

// ListIcebergRefs returns catalog refs for tenantID via GET /v4/registry/iceberg.
func (c *Client) ListIcebergRefs(ctx context.Context, tenantID string) ([]IcebergCatalogRef, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errors.New("aionclient: tenant_id required")
	}

	path := fmt.Sprintf("/v4/registry/iceberg?tenant_id=%s", url.QueryEscape(tenantID))
	var refs []IcebergCatalogRef
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &refs); err != nil {
		return nil, err
	}
	if refs == nil {
		refs = []IcebergCatalogRef{}
	}
	return refs, nil
}