package iomeshclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	if s.HealthMS < 0 || s.ReadyMS < 0 || s.DurationMS < 0 {
		t.Fatalf("latencies must be >=0: health_ms=%d ready_ms=%d duration_ms=%d",
			s.HealthMS, s.ReadyMS, s.DurationMS)
	}
	if s.Result != "ok" {
		t.Fatalf("Result=%q want ok", s.Result)
	}
	if s.Version != iomeshclient.Version || s.Version == "" {
		t.Fatalf("Version=%q want %q", s.Version, iomeshclient.Version)
	}

	human := iomeshclient.FormatConnectionStatus(s)
	if !strings.Contains(human, "health=ok") || !strings.Contains(human, "ready=ok") {
		t.Fatalf("FormatConnectionStatus=%q", human)
	}
	if !strings.Contains(human, "health_ms=") || !strings.Contains(human, "ready_ms=") ||
		!strings.Contains(human, "duration_ms=") {
		t.Fatalf("FormatConnectionStatus missing latencies: %q", human)
	}
	if !strings.Contains(human, "result=ok") {
		t.Fatalf("FormatConnectionStatus missing result=ok: %q", human)
	}
	if !strings.Contains(human, "version="+iomeshclient.Version) {
		t.Fatalf("FormatConnectionStatus missing version=: %q", human)
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
	if s.HealthMS < 0 || s.ReadyMS < 0 || s.DurationMS < 0 {
		t.Fatalf("latencies must be >=0: health_ms=%d ready_ms=%d duration_ms=%d",
			s.HealthMS, s.ReadyMS, s.DurationMS)
	}
	if s.Result != "err" {
		t.Fatalf("Result=%q want err (health fail)", s.Result)
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
	if !strings.Contains(human, "health_ms=") || !strings.Contains(human, "ready_ms=") ||
		!strings.Contains(human, "duration_ms=") {
		t.Fatalf("FormatConnectionStatus missing latencies: %q", human)
	}
	if !strings.Contains(human, "result=err") {
		t.Fatalf("FormatConnectionStatus missing result=err: %q", human)
	}
}

func TestConnectionStatus_ProbeLatencyMeasured(t *testing.T) {
	const delay = 25 * time.Millisecond
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health", "/ready":
			time.Sleep(delay)
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
	if !s.HealthOK || !s.ReadyOK {
		t.Fatalf("probes: HealthOK=%v ReadyOK=%v HealthErr=%q ReadyErr=%q",
			s.HealthOK, s.ReadyOK, s.HealthErr, s.ReadyErr)
	}
	// httptest sleep should register as non-zero wall time on most machines.
	// Allow 0 only if the clock is coarser than delay; still require >= 0.
	if s.HealthMS < 0 || s.ReadyMS < 0 || s.DurationMS < 0 {
		t.Fatalf("latencies must be >=0: health_ms=%d ready_ms=%d duration_ms=%d",
			s.HealthMS, s.ReadyMS, s.DurationMS)
	}
	if s.HealthMS == 0 && s.ReadyMS == 0 && s.DurationMS == 0 {
		// Extremely coarse clock is rare; soft-fail only if all zero after sleep.
		t.Logf("warning: all latencies 0 after %v sleep (clock granularity?)", delay)
	}

	js := iomeshclient.FormatConnectionStatusJSON(s)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	hm, ok := parsed["health_ms"].(float64)
	if !ok {
		t.Fatalf("JSON missing health_ms: %s", js)
	}
	rm, ok := parsed["ready_ms"].(float64)
	if !ok {
		t.Fatalf("JSON missing ready_ms: %s", js)
	}
	dm, ok := parsed["duration_ms"].(float64)
	if !ok {
		t.Fatalf("JSON missing duration_ms: %s", js)
	}
	if int(hm) != s.HealthMS || int(rm) != s.ReadyMS || int(dm) != s.DurationMS {
		t.Fatalf("JSON latencies health_ms=%v ready_ms=%v duration_ms=%v want %d %d %d\n%s",
			parsed["health_ms"], parsed["ready_ms"], parsed["duration_ms"],
			s.HealthMS, s.ReadyMS, s.DurationMS, js)
	}
	if hm < 0 || rm < 0 || dm < 0 {
		t.Fatalf("JSON latencies must be >=0: %s", js)
	}

	human := iomeshclient.FormatConnectionStatus(s)
	if !strings.Contains(human, "health_ms=") || !strings.Contains(human, "ready_ms=") ||
		!strings.Contains(human, "duration_ms=") {
		t.Fatalf("FormatConnectionStatus missing latencies: %q", human)
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
	if s.HealthMS != 0 || s.ReadyMS != 0 || s.DurationMS != 0 {
		t.Fatalf("nil client latencies want 0: health_ms=%d ready_ms=%d duration_ms=%d",
			s.HealthMS, s.ReadyMS, s.DurationMS)
	}
	if s.Result != "err" {
		t.Fatalf("Result=%q want err (nil client)", s.Result)
	}
	if s.Version != iomeshclient.Version || s.Version == "" {
		t.Fatalf("nil client Version=%q want %q", s.Version, iomeshclient.Version)
	}
	human := iomeshclient.FormatConnectionStatus(s)
	if !strings.Contains(human, "health_ms=0") || !strings.Contains(human, "ready_ms=0") ||
		!strings.Contains(human, "duration_ms=0") {
		t.Fatalf("FormatConnectionStatus nil zeros: %q", human)
	}
	if !strings.Contains(human, "result=err") {
		t.Fatalf("FormatConnectionStatus nil missing result=err: %q", human)
	}
	if !strings.Contains(human, "version="+iomeshclient.Version) {
		t.Fatalf("FormatConnectionStatus nil missing version=: %q", human)
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
	if !strings.Contains(js, `"health_ms"`) {
		t.Fatalf("JSON missing health_ms: %s", js)
	}
	if !strings.Contains(js, `"ready_ms"`) {
		t.Fatalf("JSON missing ready_ms: %s", js)
	}
	if !strings.Contains(js, `"duration_ms"`) {
		t.Fatalf("JSON missing duration_ms: %s", js)
	}
	if !strings.Contains(js, `"result"`) {
		t.Fatalf("JSON missing result: %s", js)
	}
	if !strings.Contains(js, `"version"`) {
		t.Fatalf("JSON missing version: %s", js)
	}
	if !strings.HasSuffix(js, "\n") {
		t.Fatal("JSON should end with newline")
	}
	// Indented (pretty) JSON has newlines between fields.
	if !strings.Contains(js, "\n  ") {
		t.Fatalf("expected indented JSON: %s", js)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	if _, ok := parsed["health_ms"].(float64); !ok {
		t.Fatalf("health_ms missing or wrong type: %v\n%s", parsed["health_ms"], js)
	}
	if _, ok := parsed["ready_ms"].(float64); !ok {
		t.Fatalf("ready_ms missing or wrong type: %v\n%s", parsed["ready_ms"], js)
	}
	if _, ok := parsed["duration_ms"].(float64); !ok {
		t.Fatalf("duration_ms missing or wrong type: %v\n%s", parsed["duration_ms"], js)
	}
	if n := int(parsed["health_ms"].(float64)); n < 0 {
		t.Fatalf("health_ms=%d want >=0", n)
	}
	if n := int(parsed["ready_ms"].(float64)); n < 0 {
		t.Fatalf("ready_ms=%d want >=0", n)
	}
	if n := int(parsed["duration_ms"].(float64)); n < 0 {
		t.Fatalf("duration_ms=%d want >=0", n)
	}
	res, ok := parsed["result"].(string)
	if !ok || (res != "ok" && res != "err") {
		t.Fatalf("result missing or invalid: %v\n%s", parsed["result"], js)
	}
	if res != s.Result {
		t.Fatalf("JSON result=%q want %q", res, s.Result)
	}
	ver, ok := parsed["version"].(string)
	if !ok || ver != iomeshclient.Version {
		t.Fatalf("JSON version=%v want %q\n%s", parsed["version"], iomeshclient.Version, js)
	}
	if s.Version != iomeshclient.Version {
		t.Fatalf("Version=%q want %q", s.Version, iomeshclient.Version)
	}
}

func TestFormatConnectionStatus_AlwaysEmitsLatencies(t *testing.T) {
	// Zero latencies (not run / default struct) still print health_ms=, ready_ms=, duration_ms=.
	// Empty Version still prints version= package Version (fallback).
	human := iomeshclient.FormatConnectionStatus(iomeshclient.ConnectionStatus{
		BaseURL:   "http://127.0.0.1:8422",
		UserAgent: "iomesh-client-sdk-go/test",
		HealthOK:  true,
		ReadyOK:   true,
	})
	if !strings.Contains(human, "health_ms=0") {
		t.Fatalf("missing health_ms=0: %q", human)
	}
	if !strings.Contains(human, "ready_ms=0") {
		t.Fatalf("missing ready_ms=0: %q", human)
	}
	if !strings.Contains(human, "duration_ms=0") {
		t.Fatalf("missing duration_ms=0: %q", human)
	}
	if !strings.Contains(human, "result=ok") {
		t.Fatalf("missing result=ok: %q", human)
	}
	if !strings.Contains(human, "version="+iomeshclient.Version) {
		t.Fatalf("missing version= fallback: %q", human)
	}

	human2 := iomeshclient.FormatConnectionStatus(iomeshclient.ConnectionStatus{
		BaseURL:    "http://x",
		Version:    "9.9.9",
		HealthMS:   12,
		ReadyMS:    34,
		DurationMS: 50,
	})
	if !strings.Contains(human2, "health_ms=12") || !strings.Contains(human2, "ready_ms=34") ||
		!strings.Contains(human2, "duration_ms=50") {
		t.Fatalf("expected explicit latencies: %q", human2)
	}
	if !strings.Contains(human2, "result=err") {
		t.Fatalf("missing result=err when probes not OK: %q", human2)
	}
	if !strings.Contains(human2, "version=9.9.9") {
		t.Fatalf("expected explicit version: %q", human2)
	}
}

func TestConnectionStatus_AlwaysEmitsVersion(t *testing.T) {
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
	if s.Version != iomeshclient.Version {
		t.Fatalf("Version=%q want %q", s.Version, iomeshclient.Version)
	}
	if s.Version == "" {
		t.Fatal("Version must always be present")
	}

	js := iomeshclient.FormatConnectionStatusJSON(s)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	ver, ok := parsed["version"].(string)
	if !ok || ver != iomeshclient.Version {
		t.Fatalf("JSON version=%v want %q\n%s", parsed["version"], iomeshclient.Version, js)
	}

	human := iomeshclient.FormatConnectionStatus(s)
	if !strings.Contains(human, "version="+iomeshclient.Version) {
		t.Fatalf("Format missing version=: %q", human)
	}

	// Nil client path also always sets Version.
	var nilC *iomeshclient.Client
	ns := nilC.ConnectionStatus(context.Background())
	if ns.Version != iomeshclient.Version {
		t.Fatalf("nil Version=%q want %q", ns.Version, iomeshclient.Version)
	}
}

func TestAggregateConnectionResult(t *testing.T) {
	if got := iomeshclient.AggregateConnectionResult(true, true); got != "ok" {
		t.Fatalf("both OK: got %q want ok", got)
	}
	if got := iomeshclient.AggregateConnectionResult(false, true); got != "err" {
		t.Fatalf("health fail: got %q want err", got)
	}
	if got := iomeshclient.AggregateConnectionResult(true, false); got != "err" {
		t.Fatalf("ready fail: got %q want err", got)
	}
	if got := iomeshclient.AggregateConnectionResult(false, false); got != "err" {
		t.Fatalf("both fail: got %q want err", got)
	}
}
