package iomeshclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
)

func TestConnectionStatus_HealthAndReadyOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health", "/ready":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL},
		iomeshclient.WithTenant("t1"),
		iomeshclient.WithOrg("o1"),
		iomeshclient.WithWorkspace("w1"),
	)
	if err != nil {
		t.Fatal(err)
	}

	s := nc.ConnectionStatus(context.Background())
	if s.BaseURL != srv.URL {
		t.Fatalf("BaseURL=%q want %q", s.BaseURL, srv.URL)
	}
	if s.Tenant != "t1" || s.Org != "o1" || s.Workspace != "w1" {
		t.Fatalf("identity tenant=%q org=%q workspace=%q", s.Tenant, s.Org, s.Workspace)
	}
	if !strings.HasPrefix(s.UserAgent, "iomesh-client-sdk-go/") {
		t.Fatalf("UserAgent=%q", s.UserAgent)
	}
	if !s.HealthOK || s.HealthErr != "" {
		t.Fatalf("HealthOK=%v HealthErr=%q", s.HealthOK, s.HealthErr)
	}
	if !s.ReadyOK || s.ReadyErr != "" {
		t.Fatalf("ReadyOK=%v ReadyErr=%q", s.ReadyOK, s.ReadyErr)
	}

	human := iomeshclient.FormatConnectionStatus(s)
	if !strings.Contains(human, "health=ok") || !strings.Contains(human, "ready=ok") {
		t.Fatalf("FormatConnectionStatus=%q", human)
	}
}

func TestConnectionStatus_HealthFailReadyOK(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusServiceUnavailable)
		case "/ready":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}

	s := nc.ConnectionStatus(context.Background())
	if s.HealthOK {
		t.Fatal("expected HealthOK=false")
	}
	if s.HealthErr == "" || !strings.Contains(s.HealthErr, "503") {
		t.Fatalf("HealthErr=%q", s.HealthErr)
	}
	if !s.ReadyOK || s.ReadyErr != "" {
		t.Fatalf("ReadyOK=%v ReadyErr=%q", s.ReadyOK, s.ReadyErr)
	}
	// Both probes must run even when Health fails.
	hasHealth, hasReady := false, false
	for _, p := range paths {
		if p == "/health" {
			hasHealth = true
		}
		if p == "/ready" || p == "/readyz" {
			hasReady = true
		}
	}
	if !hasHealth || !hasReady {
		t.Fatalf("paths=%v (want both health and ready probes)", paths)
	}

	human := iomeshclient.FormatConnectionStatus(s)
	if !strings.Contains(human, "health=FAIL") || !strings.Contains(human, "ready=ok") {
		t.Fatalf("FormatConnectionStatus=%q", human)
	}
}

func TestConnectionStatus_NilClient(t *testing.T) {
	var c *iomeshclient.Client
	s := c.ConnectionStatus(context.Background())
	if s.HealthOK || s.ReadyOK {
		t.Fatalf("expected both probes failed: %+v", s)
	}
	if s.HealthErr != "nil client" || s.ReadyErr != "nil client" {
		t.Fatalf("HealthErr=%q ReadyErr=%q", s.HealthErr, s.ReadyErr)
	}
	if s.BaseURL != "" || s.UserAgent != "" {
		t.Fatalf("expected empty identity fields: %+v", s)
	}
}

func TestConnectionStatus_JSONContainsBaseURLAndUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health", "/ready":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	s := nc.ConnectionStatus(context.Background())
	js := iomeshclient.FormatConnectionStatusJSON(s)
	if !strings.Contains(js, `"base_url"`) {
		t.Fatalf("JSON missing base_url: %s", js)
	}
	if !strings.Contains(js, `"user_agent"`) {
		t.Fatalf("JSON missing user_agent: %s", js)
	}
	if !strings.HasSuffix(js, "\n") {
		t.Fatal("JSON should end with newline")
	}
	// Indented (pretty) JSON has newlines between fields.
	if !strings.Contains(js, "\n  ") {
		t.Fatalf("expected indented JSON: %s", js)
	}
}
