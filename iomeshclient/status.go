package iomeshclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ConnectionStatus is a fail-open snapshot of client identity + probes.
// Fields are always populated for operators/CI; probe failures set *OK=false and *Err.
type ConnectionStatus struct {
	BaseURL   string `json:"base_url"`
	Tenant    string `json:"tenant,omitempty"`
	Org       string `json:"org,omitempty"`
	Workspace string `json:"workspace,omitempty"`
	UserAgent string `json:"user_agent"`
	// Version is the SDK package version (always emitted; equals iomeshclient.Version, including nil client).
	Version   string `json:"version"`
	HealthOK  bool   `json:"health_ok"`
	HealthErr string `json:"health_err,omitempty"`
	// HealthMS is Health probe latency in milliseconds (always emitted; 0 when nil client / not run).
	HealthMS int    `json:"health_ms"`
	ReadyOK  bool   `json:"ready_ok"`
	ReadyErr string `json:"ready_err,omitempty"`
	// ReadyMS is Ready probe latency in milliseconds (always emitted; 0 when nil client / not run).
	ReadyMS int `json:"ready_ms"`
	// DurationMS is wall-clock latency for the full Health+Ready probe path in milliseconds
	// (always emitted; 0 when nil client / not run).
	DurationMS int `json:"duration_ms"`
	// Result is the aggregate probe outcome: "ok" when both HealthOK and ReadyOK are true,
	// otherwise "err" (always emitted; includes nil client and either-probe fail).
	Result string `json:"result"`
}

// AggregateConnectionResult returns "ok" when both probes succeeded, otherwise "err".
// Pure helper for operators/CI and mesh status result parity.
func AggregateConnectionResult(healthOK, readyOK bool) string {
	if healthOK && readyOK {
		return "ok"
	}
	return "err"
}

// elapsedMS converts a duration to non-negative milliseconds for probe evidence.
func elapsedMS(d time.Duration) int {
	ms := int(d.Milliseconds())
	if ms < 0 {
		return 0
	}
	return ms
}

// ConnectionStatus probes Health then Ready (fail-open fields; never panics).
// Nil client → empty with HealthErr/ReadyErr "nil client" (HealthMS/ReadyMS/DurationMS stay 0; Result "err").
// Version is always set to the package Version constant (including nil client).
// Does not short-circuit Ready when Health fails — both probes always run.
// Probe wall times are always set as HealthMS / ReadyMS / DurationMS (>= 0).
// DurationMS is wall clock for the full Health+Ready path (start before Health, stop after Ready).
// Result is always "ok" | "err" (both probes OK → "ok"; otherwise "err").
func (c *Client) ConnectionStatus(ctx context.Context) ConnectionStatus {
	if c == nil {
		return ConnectionStatus{
			Version:   Version,
			HealthErr: "nil client",
			ReadyErr:  "nil client",
			Result:    AggregateConnectionResult(false, false),
		}
	}
	if ctx == nil {
		ctx = context.Background()
	}

	s := ConnectionStatus{
		BaseURL:   c.baseURL,
		Tenant:    c.tenant,
		Org:       c.org,
		Workspace: c.workspace,
		UserAgent: c.userAgent,
		Version:   Version,
	}
	if s.UserAgent == "" {
		s.UserAgent = defaultUserAgent
	}

	start := time.Now()

	t0 := time.Now()
	if err := c.Health(ctx); err != nil {
		s.HealthOK = false
		s.HealthErr = err.Error()
	} else {
		s.HealthOK = true
	}
	s.HealthMS = elapsedMS(time.Since(t0))

	t1 := time.Now()
	if err := c.Ready(ctx); err != nil {
		s.ReadyOK = false
		s.ReadyErr = err.Error()
	} else {
		s.ReadyOK = true
	}
	s.ReadyMS = elapsedMS(time.Since(t1))

	s.DurationMS = elapsedMS(time.Since(start))
	s.Result = AggregateConnectionResult(s.HealthOK, s.ReadyOK)

	return s
}

// FormatConnectionStatus returns a human multi-line summary of ConnectionStatus.
// Always emits version, health_ms, ready_ms, duration_ms (including 0), and result=ok|err.
func FormatConnectionStatus(s ConnectionStatus) string {
	var b strings.Builder
	fmt.Fprintf(&b, "base_url=%s\n", s.BaseURL)
	if s.Tenant != "" {
		fmt.Fprintf(&b, "tenant=%s\n", s.Tenant)
	}
	if s.Org != "" {
		fmt.Fprintf(&b, "org=%s\n", s.Org)
	}
	if s.Workspace != "" {
		fmt.Fprintf(&b, "workspace=%s\n", s.Workspace)
	}
	fmt.Fprintf(&b, "user_agent=%s\n", s.UserAgent)
	ver := s.Version
	if ver == "" {
		ver = Version
	}
	fmt.Fprintf(&b, "version=%s\n", ver)
	if s.HealthOK {
		b.WriteString("health=ok\n")
	} else {
		fmt.Fprintf(&b, "health=FAIL")
		if s.HealthErr != "" {
			fmt.Fprintf(&b, " err=%s", s.HealthErr)
		}
		b.WriteByte('\n')
	}
	fmt.Fprintf(&b, "health_ms=%d\n", s.HealthMS)
	if s.ReadyOK {
		b.WriteString("ready=ok\n")
	} else {
		fmt.Fprintf(&b, "ready=FAIL")
		if s.ReadyErr != "" {
			fmt.Fprintf(&b, " err=%s", s.ReadyErr)
		}
		b.WriteByte('\n')
	}
	fmt.Fprintf(&b, "ready_ms=%d\n", s.ReadyMS)
	fmt.Fprintf(&b, "duration_ms=%d\n", s.DurationMS)
	result := s.Result
	if result != "ok" && result != "err" {
		result = AggregateConnectionResult(s.HealthOK, s.ReadyOK)
	}
	fmt.Fprintf(&b, "result=%s\n", result)
	return b.String()
}

// FormatConnectionStatusJSON returns indented JSON of ConnectionStatus plus a trailing newline.
func FormatConnectionStatusJSON(s ConnectionStatus) string {
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		// ConnectionStatus is simple value types; marshal failure is unexpected.
		return "{}\n"
	}
	return string(raw) + "\n"
}
