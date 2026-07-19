package iomeshclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ConnectionStatus is a fail-open snapshot of client identity + probes.
// Fields are always populated for operators/CI; probe failures set *OK=false and *Err.
type ConnectionStatus struct {
	BaseURL   string `json:"base_url"`
	Tenant    string `json:"tenant,omitempty"`
	Org       string `json:"org,omitempty"`
	Workspace string `json:"workspace,omitempty"`
	UserAgent string `json:"user_agent"`
	HealthOK  bool   `json:"health_ok"`
	HealthErr string `json:"health_err,omitempty"`
	ReadyOK   bool   `json:"ready_ok"`
	ReadyErr  string `json:"ready_err,omitempty"`
}

// ConnectionStatus probes Health then Ready (fail-open fields; never panics).
// Nil client → empty with HealthErr/ReadyErr "nil client".
// Does not short-circuit Ready when Health fails — both probes always run.
func (c *Client) ConnectionStatus(ctx context.Context) ConnectionStatus {
	if c == nil {
		return ConnectionStatus{
			HealthErr: "nil client",
			ReadyErr:  "nil client",
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
	}
	if s.UserAgent == "" {
		s.UserAgent = defaultUserAgent
	}

	if err := c.Health(ctx); err != nil {
		s.HealthOK = false
		s.HealthErr = err.Error()
	} else {
		s.HealthOK = true
	}

	if err := c.Ready(ctx); err != nil {
		s.ReadyOK = false
		s.ReadyErr = err.Error()
	} else {
		s.ReadyOK = true
	}

	return s
}

// FormatConnectionStatus returns a human multi-line summary of ConnectionStatus.
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
	if s.HealthOK {
		b.WriteString("health=ok\n")
	} else {
		fmt.Fprintf(&b, "health=FAIL")
		if s.HealthErr != "" {
			fmt.Fprintf(&b, " err=%s", s.HealthErr)
		}
		b.WriteByte('\n')
	}
	if s.ReadyOK {
		b.WriteString("ready=ok\n")
	} else {
		fmt.Fprintf(&b, "ready=FAIL")
		if s.ReadyErr != "" {
			fmt.Fprintf(&b, " err=%s", s.ReadyErr)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// FormatConnectionStatusJSON returns indented JSON of ConnectionStatus plus a trailing newline.
func FormatConnectionStatusJSON(s ConnectionStatus) string {
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		// ConnectionStatus is all strings/bools; marshal failure is unexpected.
		return "{}\n"
	}
	return string(raw) + "\n"
}
