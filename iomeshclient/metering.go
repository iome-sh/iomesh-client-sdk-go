package iomeshclient

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Stream name for operational dept.* events (POST /v1/streams/dept/publish).
const streamDept = "dept"

// DeptEvent is a lightweight operational stream event (dept.* family).
// Wire shape matches iomesh-tui internal/iomesh.DeptEvent for platform remote metering.
type DeptEvent struct {
	Type      string         `json:"type"` // e.g. dept.agent.llm_call
	Timestamp time.Time      `json:"ts"`
	Tenant    string         `json:"tenant,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
}

// LLMCallEvent is the structured payload for dept.agent.llm_call (remote metering dashboards).
type LLMCallEvent struct {
	Tenant    string
	SessionID string
	Model     string
	ModelID   string
	// Org / Workspace are also set as Connect headers (WithOrg / WithWorkspace);
	// fields here mirror iomesh-tui RecordLLMCall payload for consumers that only read the body.
	Org              string
	Workspace        string
	DurationMS       int64
	Attempts         int
	Fallback         bool
	EstUSD           float64
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	Error            string // optional redacted error string
	Extra            map[string]any
}

// EmitDeptEvent publishes a structured dept.* event via POST /v1/streams/dept/publish.
// Subject defaults to ev.Type (e.g. dept.agent.llm_call). Tenant defaults to WithTenant.
func (c *Client) EmitDeptEvent(ctx context.Context, ev DeptEvent) (*PubAck, error) {
	if c == nil {
		return nil, errors.New("iomeshclient: nil client")
	}
	if strings.TrimSpace(ev.Type) == "" {
		return nil, errors.New("iomeshclient: type required")
	}
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}
	if strings.TrimSpace(ev.Tenant) == "" {
		ev.Tenant = c.tenant
	}
	if ev.Payload == nil {
		ev.Payload = map[string]any{}
	}
	// Enrich multi-tenant fields when configured on the client (parity with iomesh-tui).
	if org := strings.TrimSpace(c.org); org != "" {
		if _, ok := ev.Payload["org"]; !ok {
			ev.Payload["org"] = org
		}
	}
	if ws := strings.TrimSpace(c.workspace); ws != "" {
		if _, ok := ev.Payload["workspace"]; !ok {
			ev.Payload["workspace"] = ws
		}
	}
	if t := strings.TrimSpace(ev.Tenant); t != "" {
		if _, ok := ev.Payload["tenant"]; !ok {
			ev.Payload["tenant"] = t
		}
	}
	raw, err := json.Marshal(ev)
	if err != nil {
		return nil, err
	}
	subject := strings.TrimSpace(ev.Type)
	return c.Publish(ctx, streamDept, subject, raw)
}

// EmitLLMCall publishes dept.agent.llm_call for platform remote metering dashboards.
// Uses the same multi-tenant headers as other client methods (WithOrg / WithWorkspace).
func (c *Client) EmitLLMCall(ctx context.Context, call LLMCallEvent) (*PubAck, error) {
	if c == nil {
		return nil, errors.New("iomeshclient: nil client")
	}
	payload := map[string]any{
		"model":       call.Model,
		"model_id":    call.ModelID,
		"duration_ms": call.DurationMS,
		"attempts":    call.Attempts,
		"fallback":    call.Fallback,
		"est_usd":     call.EstUSD,
		"tokens": map[string]int{
			"prompt":     call.PromptTokens,
			"completion": call.CompletionTokens,
			"total":      call.TotalTokens,
		},
	}
	if t := strings.TrimSpace(call.Tenant); t != "" {
		payload["tenant"] = t
	}
	if org := strings.TrimSpace(call.Org); org != "" {
		payload["org"] = org
	}
	if ws := strings.TrimSpace(call.Workspace); ws != "" {
		payload["workspace"] = ws
	}
	if errMsg := strings.TrimSpace(call.Error); errMsg != "" {
		payload["error"] = errMsg
	}
	for k, v := range call.Extra {
		if k == "" {
			continue
		}
		payload[k] = v
	}
	return c.EmitDeptEvent(ctx, DeptEvent{
		Type:      "dept.agent.llm_call",
		Tenant:    call.Tenant,
		SessionID: call.SessionID,
		Payload:   payload,
	})
}
