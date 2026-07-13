package aionclient

import (
	"context"
	"errors"
	"net/http"
)

// SubsetConfig describes a virtual subset stream over a source stream.
type SubsetConfig struct {
	ID            string   `json:"id"`
	SourceStream  string   `json:"source_stream"`
	SubjectFilter string   `json:"subject_filter"`
	FieldMask     []string `json:"field_mask,omitempty"`
	Tenant        string   `json:"tenant"`
}

// CreateSubset registers a subset stream via POST /v2/subsets.
func (c *Client) CreateSubset(ctx context.Context, cfg SubsetConfig) error {
	if cfg.ID == "" {
		return errors.New("aionclient: subset id required")
	}
	if cfg.SourceStream == "" {
		return errors.New("aionclient: source_stream required")
	}
	if cfg.SubjectFilter == "" {
		return errors.New("aionclient: subject_filter required")
	}
	if cfg.Tenant == "" {
		return errors.New("aionclient: tenant required")
	}
	return c.doJSON(ctx, http.MethodPost, "/v2/subsets", cfg, new(struct{}))
}