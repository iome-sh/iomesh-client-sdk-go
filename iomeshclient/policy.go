package iomeshclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// PolicyMode controls remote policy evaluation for agent tools.
// Values: off | advisory | enforce.
type PolicyMode string

const (
	PolicyOff      PolicyMode = "off"
	PolicyAdvisory PolicyMode = "advisory"
	PolicyEnforce  PolicyMode = "enforce"
)

// PolicyInput is sent to POST /v1/policy/evaluate.
// Mode is per-call (this SDK has no offline policy config); empty/off → Source=off, Allow=true.
type PolicyInput struct {
	Action     string
	Resource   string
	Tool       string
	Attributes map[string]any
	Mode       PolicyMode
}

// PolicyDecision is the evaluate response (broker Rego / OPA path).
type PolicyDecision struct {
	Allow   bool
	Reasons []string
	// Source describes how the decision was reached (mesh | fail-open | off | unavailable).
	Source string
	// Mode echoes the (normalized) mode used for this check.
	Mode PolicyMode
}

// normalizePolicyMode lowercases/trims; only advisory/enforce stick, else off.
func normalizePolicyMode(m PolicyMode) PolicyMode {
	switch PolicyMode(strings.ToLower(strings.TrimSpace(string(m)))) {
	case PolicyAdvisory:
		return PolicyAdvisory
	case PolicyEnforce:
		return PolicyEnforce
	default:
		return PolicyOff
	}
}

// EvaluatePolicy POSTs to {base}/v1/policy/evaluate.
//
// Semantics (iomesh-tui parity, without dept emit side-effect):
//   - Mode off or empty → Allow true, Source off (no network)
//   - nil client → if Mode off: Source off; else fail-open with reason "nil client"
//   - Empty Action with Tool set → Action = "tool."+Tool
//   - 404 → Allow true, Source unavailable
//   - transport / non-OK / decode errors → fail-open (Allow true, Source fail-open)
//   - mesh success → Source mesh; decode allow/allowed/deny/reason/reasons
//
// Enforce mode only blocks via ShouldBlockTool when mesh explicitly denies.
// This helper never auto-emits dept audit events.
func (c *Client) EvaluatePolicy(ctx context.Context, in PolicyInput) PolicyDecision {
	mode := normalizePolicyMode(in.Mode)
	if mode == PolicyOff {
		return PolicyDecision{Allow: true, Source: "off", Mode: mode}
	}
	if c == nil {
		return PolicyDecision{
			Allow: true, Source: "fail-open", Mode: mode,
			Reasons: []string{"nil client"},
		}
	}

	if strings.TrimSpace(in.Action) == "" && in.Tool != "" {
		in.Action = "tool." + in.Tool
	}

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	payload := map[string]any{
		"tenant":     c.tenant,
		"action":     in.Action,
		"resource":   in.Resource,
		"tool":       in.Tool,
		"attributes": in.Attributes,
		"mode":       string(mode),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return PolicyDecision{
			Allow: true, Source: "fail-open", Mode: mode,
			Reasons: []string{"marshal: " + err.Error()},
		}
	}

	url := c.baseURL + "/v1/policy/evaluate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return PolicyDecision{
			Allow: true, Source: "fail-open", Mode: mode,
			Reasons: []string{err.Error()},
		}
	}
	c.applyAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return PolicyDecision{
			Allow: true, Source: "fail-open", Mode: mode,
			Reasons: []string{err.Error()},
		}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return PolicyDecision{
			Allow: true, Source: "unavailable", Mode: mode,
			Reasons: []string{"policy endpoint 404"},
		}
	}
	if resp.StatusCode != http.StatusOK {
		return PolicyDecision{
			Allow: true, Source: "fail-open", Mode: mode,
			Reasons: []string{fmt.Sprintf("http %d", resp.StatusCode)},
		}
	}

	var out struct {
		Allow   *bool    `json:"allow"`
		Allowed *bool    `json:"allowed"`
		Reasons []string `json:"reasons"`
		Reason  string   `json:"reason"`
		Deny    bool     `json:"deny"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return PolicyDecision{
			Allow: true, Source: "fail-open", Mode: mode,
			Reasons: []string{"decode: " + err.Error()},
		}
	}

	allow := true
	if out.Allow != nil {
		allow = *out.Allow
	} else if out.Allowed != nil {
		allow = *out.Allowed
	} else if out.Deny {
		allow = false
	}
	reasons := out.Reasons
	if out.Reason != "" {
		reasons = append(reasons, out.Reason)
	}
	return PolicyDecision{Allow: allow, Reasons: reasons, Source: "mesh", Mode: mode}
}

// ShouldBlockTool returns true only when mode is enforce and mesh explicitly denies.
func (d PolicyDecision) ShouldBlockTool() bool {
	return d.Mode == PolicyEnforce && !d.Allow && d.Source == "mesh"
}

// Summary is a short operator-facing string.
func (d PolicyDecision) Summary() string {
	if d.Allow {
		return fmt.Sprintf("allow source=%s mode=%s", d.Source, d.Mode)
	}
	r := strings.Join(d.Reasons, "; ")
	if r == "" {
		r = "denied"
	}
	return fmt.Sprintf("deny source=%s mode=%s reasons=%s", d.Source, d.Mode, r)
}
