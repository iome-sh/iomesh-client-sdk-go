package iomeshclient

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// LineageRef is a governed data-product / stream lineage pointer from the context plane.
type LineageRef struct {
	ID        string `json:"id,omitempty"`
	Product   string `json:"product,omitempty"`
	Subject   string `json:"subject,omitempty"`
	Source    string `json:"source,omitempty"`
	Freshness string `json:"freshness,omitempty"`
}

// ContextResult is a fail-open context plane response (text + optional lineage).
type ContextResult struct {
	Text    string
	Lineage []LineageRef
}

// QueryContextRequest is the body for POST /v1/context/query.
// Limit defaults to 20 when <= 0. IncludeLineage is opt-in on the wire;
// ContextSnippet always sets it true for agent prompt injection.
type QueryContextRequest struct {
	Workspace      string
	Query          string
	Limit          int // default 20 if <= 0
	IncludeLineage bool
}

// QueryContext POSTs to {base}/v1/context/query.
// Fail-open: nil client or any error (transport, non-OK, decode) → empty ContextResult.
// Empty query still POSTs (broker may return empty text); no offline ContextPlane flag.
func (c *Client) QueryContext(ctx context.Context, req QueryContextRequest) ContextResult {
	var empty ContextResult
	if c == nil {
		return empty
	}

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	payload := map[string]any{
		"tenant":    c.tenant,
		"workspace": req.Workspace,
		"query":     req.Query,
		"limit":     limit,
	}
	if req.IncludeLineage {
		payload["include_lineage"] = true
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return empty
	}

	url := c.baseURL + "/v1/context/query"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return empty
	}
	c.applyAuthHeaders(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return empty
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return empty
	}

	var out struct {
		Text    string       `json:"text"`
		Lineage []LineageRef `json:"lineage"`
		// Alternate shapes used by some brokers.
		Items []struct {
			Text    string       `json:"text"`
			Lineage []LineageRef `json:"lineage"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return empty
	}
	res := ContextResult{Text: strings.TrimSpace(out.Text), Lineage: out.Lineage}
	if res.Text == "" && len(out.Items) > 0 {
		var parts []string
		for _, it := range out.Items {
			if t := strings.TrimSpace(it.Text); t != "" {
				parts = append(parts, t)
			}
			res.Lineage = append(res.Lineage, it.Lineage...)
		}
		res.Text = strings.Join(parts, "\n")
	}
	return res
}

// ContextSnippet is fail-open prompt injection text for agent system prompts.
// Always requests IncludeLineage=true (agent default). Errors → empty string.
func (c *Client) ContextSnippet(ctx context.Context, workspace, query string) string {
	return FormatContextSnippet(c.QueryContext(ctx, QueryContextRequest{
		Workspace:      workspace,
		Query:          query,
		IncludeLineage: true,
	}))
}

// FormatContextSnippet merges text + lineage for prompt injection.
// Lineage is rendered as a compact <iomesh-lineage> block (max 12 refs).
func FormatContextSnippet(res ContextResult) string {
	var b strings.Builder
	if t := strings.TrimSpace(res.Text); t != "" {
		b.WriteString(t)
	}
	if len(res.Lineage) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("<iomesh-lineage>\n")
		for i, ref := range res.Lineage {
			if i >= 12 {
				b.WriteString("…\n")
				break
			}
			id := firstNonEmpty(ref.ID, ref.Product)
			parts := make([]string, 0, 4)
			if id != "" {
				parts = append(parts, id)
			}
			if ref.Subject != "" {
				parts = append(parts, "subject="+ref.Subject)
			}
			if ref.Source != "" {
				parts = append(parts, "source="+ref.Source)
			}
			if ref.Freshness != "" {
				parts = append(parts, "freshness="+ref.Freshness)
			}
			if len(parts) == 0 {
				continue
			}
			b.WriteString("- ")
			b.WriteString(strings.Join(parts, " · "))
			b.WriteByte('\n')
		}
		b.WriteString("</iomesh-lineage>")
	}
	return strings.TrimSpace(b.String())
}
