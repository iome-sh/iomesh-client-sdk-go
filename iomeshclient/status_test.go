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
	if !strings.Contains(human, "health_err=\n") || !strings.Contains(human, "ready_err=\n") {
		t.Fatalf("FormatConnectionStatus missing empty probe errs: %q", human)
	}
	if !strings.Contains(human, "tenant=t1") || !strings.Contains(human, "org=o1") ||
		!strings.Contains(human, "workspace=w1") {
		t.Fatalf("FormatConnectionStatus missing identity: %q", human)
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
	if !strings.Contains(human, "health=FAIL\n") || !strings.Contains(human, "ready=ok") {
		t.Fatalf("FormatConnectionStatus=%q", human)
	}
	if !strings.Contains(human, "health_err=") || !strings.Contains(human, s.HealthErr) {
		t.Fatalf("FormatConnectionStatus missing health_err detail: %q", human)
	}
	if !strings.Contains(human, "ready_err=\n") {
		t.Fatalf("FormatConnectionStatus missing empty ready_err=: %q", human)
	}
	// FAIL lines must not inline err= (detail lives on health_err=/ready_err= only).
	if strings.Contains(human, "health=FAIL err=") || strings.Contains(human, "ready=FAIL err=") {
		t.Fatalf("FormatConnectionStatus must not inline err= on status lines: %q", human)
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
	if s.Tenant != "" || s.Org != "" || s.Workspace != "" {
		t.Fatalf("nil client identity want empty: tenant=%q org=%q workspace=%q",
			s.Tenant, s.Org, s.Workspace)
	}
	human := iomeshclient.FormatConnectionStatus(s)
	if !strings.Contains(human, "health_ms=0") || !strings.Contains(human, "ready_ms=0") ||
		!strings.Contains(human, "duration_ms=0") {
		t.Fatalf("FormatConnectionStatus nil zeros: %q", human)
	}
	if !strings.Contains(human, "health=FAIL\n") || !strings.Contains(human, "ready=FAIL\n") {
		t.Fatalf("FormatConnectionStatus nil missing FAIL status: %q", human)
	}
	if !strings.Contains(human, "health_err=nil client\n") || !strings.Contains(human, "ready_err=nil client\n") {
		t.Fatalf("FormatConnectionStatus nil missing probe errs: %q", human)
	}
	if !strings.Contains(human, "tenant=\n") || !strings.Contains(human, "org=\n") ||
		!strings.Contains(human, "workspace=\n") {
		t.Fatalf("FormatConnectionStatus nil missing empty identity: %q", human)
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
	if !strings.Contains(js, `"tenant"`) {
		t.Fatalf("JSON missing tenant: %s", js)
	}
	if !strings.Contains(js, `"org"`) {
		t.Fatalf("JSON missing org: %s", js)
	}
	if !strings.Contains(js, `"workspace"`) {
		t.Fatalf("JSON missing workspace: %s", js)
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
	if !strings.Contains(js, `"health_err"`) {
		t.Fatalf("JSON missing health_err: %s", js)
	}
	if !strings.Contains(js, `"ready_err"`) {
		t.Fatalf("JSON missing ready_err: %s", js)
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
	// Identity keys always present (empty string when unset — no omitempty).
	for _, key := range []string{"tenant", "org", "workspace"} {
		v, ok := parsed[key]
		if !ok {
			t.Fatalf("JSON missing always-emitted %q: %s", key, js)
		}
		if _, isStr := v.(string); !isStr {
			t.Fatalf("JSON %q want string got %T: %s", key, v, js)
		}
	}
	if parsed["tenant"] != "" || parsed["org"] != "" || parsed["workspace"] != "" {
		t.Fatalf("JSON identity want empty when unset: tenant=%v org=%v workspace=%v",
			parsed["tenant"], parsed["org"], parsed["workspace"])
	}
	// Probe err keys always present (empty string when OK — no omitempty).
	for _, key := range []string{"health_err", "ready_err"} {
		v, ok := parsed[key]
		if !ok {
			t.Fatalf("JSON missing always-emitted %q: %s", key, js)
		}
		if _, isStr := v.(string); !isStr {
			t.Fatalf("JSON %q want string got %T: %s", key, v, js)
		}
	}
	if parsed["health_err"] != "" || parsed["ready_err"] != "" {
		t.Fatalf("JSON probe errs want empty when OK: health_err=%v ready_err=%v",
			parsed["health_err"], parsed["ready_err"])
	}
}

func TestFormatConnectionStatus_AlwaysEmitsLatencies(t *testing.T) {
	// Zero latencies (not run / default struct) still print health_ms=, ready_ms=, duration_ms=.
	// Empty identity still prints tenant= / org= / workspace=.
	// Empty probe errs still print health_err= / ready_err=.
	// Empty Version still prints version= package Version (fallback).
	human := iomeshclient.FormatConnectionStatus(iomeshclient.ConnectionStatus{
		BaseURL:   "http://127.0.0.1:8422",
		UserAgent: "iomesh-client-sdk-go/test",
		HealthOK:  true,
		ReadyOK:   true,
	})
	if !strings.Contains(human, "tenant=\n") {
		t.Fatalf("missing tenant=: %q", human)
	}
	if !strings.Contains(human, "org=\n") {
		t.Fatalf("missing org=: %q", human)
	}
	if !strings.Contains(human, "workspace=\n") {
		t.Fatalf("missing workspace=: %q", human)
	}
	if !strings.Contains(human, "health_err=\n") {
		t.Fatalf("missing health_err=: %q", human)
	}
	if !strings.Contains(human, "ready_err=\n") {
		t.Fatalf("missing ready_err=: %q", human)
	}
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
		Tenant:     "t",
		Org:        "o",
		Workspace:  "w",
		Version:    "9.9.9",
		HealthErr:  "connection refused",
		ReadyErr:   "not ready",
		HealthMS:   12,
		ReadyMS:    34,
		DurationMS: 50,
	})
	if !strings.Contains(human2, "tenant=t\n") || !strings.Contains(human2, "org=o\n") ||
		!strings.Contains(human2, "workspace=w\n") {
		t.Fatalf("expected explicit identity: %q", human2)
	}
	if !strings.Contains(human2, "health=FAIL\n") || !strings.Contains(human2, "ready=FAIL\n") {
		t.Fatalf("expected FAIL status lines: %q", human2)
	}
	if !strings.Contains(human2, "health_err=connection refused\n") ||
		!strings.Contains(human2, "ready_err=not ready\n") {
		t.Fatalf("expected explicit probe errs: %q", human2)
	}
	if strings.Contains(human2, "health=FAIL err=") || strings.Contains(human2, "ready=FAIL err=") {
		t.Fatalf("must not inline err= on status lines: %q", human2)
	}
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

func TestConnectionStatus_AlwaysEmitsProbeErrs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health", "/ready":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Probes OK: empty health_err / ready_err still present in JSON + Format.
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	s := nc.ConnectionStatus(context.Background())
	if s.HealthErr != "" || s.ReadyErr != "" {
		t.Fatalf("OK probes want empty errs: HealthErr=%q ReadyErr=%q", s.HealthErr, s.ReadyErr)
	}

	js := iomeshclient.FormatConnectionStatusJSON(s)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	for _, key := range []string{"health_err", "ready_err"} {
		v, ok := parsed[key].(string)
		if !ok {
			t.Fatalf("JSON %q missing or not string: %v\n%s", key, parsed[key], js)
		}
		if v != "" {
			t.Fatalf("JSON %q=%q want empty\n%s", key, v, js)
		}
	}

	human := iomeshclient.FormatConnectionStatus(s)
	if !strings.Contains(human, "health_err=\n") || !strings.Contains(human, "ready_err=\n") {
		t.Fatalf("Format missing empty probe err lines: %q", human)
	}
	if !strings.Contains(human, "health=ok\n") || !strings.Contains(human, "ready=ok\n") {
		t.Fatalf("Format missing ok status lines: %q", human)
	}

	// Probe fail: health_err present with detail; ready_err empty when ready OK.
	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusServiceUnavailable)
		case "/ready":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer failSrv.Close()

	nc2, err := iomeshclient.Connect(iomeshclient.Options{URL: failSrv.URL})
	if err != nil {
		t.Fatal(err)
	}
	s2 := nc2.ConnectionStatus(context.Background())
	if s2.HealthErr == "" {
		t.Fatal("health fail want non-empty HealthErr")
	}
	if s2.ReadyErr != "" {
		t.Fatalf("ready OK want empty ReadyErr: %q", s2.ReadyErr)
	}
	js2 := iomeshclient.FormatConnectionStatusJSON(s2)
	var parsed2 map[string]any
	if err := json.Unmarshal([]byte(js2), &parsed2); err != nil {
		t.Fatalf("json2: %v\n%s", err, js2)
	}
	he, ok := parsed2["health_err"].(string)
	if !ok || he == "" {
		t.Fatalf("JSON health_err missing or empty: %v\n%s", parsed2["health_err"], js2)
	}
	if he != s2.HealthErr {
		t.Fatalf("JSON health_err=%q want %q", he, s2.HealthErr)
	}
	re, ok := parsed2["ready_err"].(string)
	if !ok {
		t.Fatalf("JSON ready_err missing: %v\n%s", parsed2["ready_err"], js2)
	}
	if re != "" {
		t.Fatalf("JSON ready_err=%q want empty", re)
	}
	human2 := iomeshclient.FormatConnectionStatus(s2)
	if !strings.Contains(human2, "health=FAIL\n") {
		t.Fatalf("Format missing health=FAIL: %q", human2)
	}
	if !strings.Contains(human2, "health_err="+s2.HealthErr+"\n") {
		t.Fatalf("Format missing health_err detail: %q", human2)
	}
	if !strings.Contains(human2, "ready_err=\n") {
		t.Fatalf("Format missing empty ready_err=: %q", human2)
	}
	if strings.Contains(human2, "health=FAIL err=") {
		t.Fatalf("must not inline err= on health line: %q", human2)
	}

	// Nil client: both errs "nil client" always present in JSON + Format.
	var nilC *iomeshclient.Client
	ns := nilC.ConnectionStatus(context.Background())
	if ns.HealthErr != "nil client" || ns.ReadyErr != "nil client" {
		t.Fatalf("nil errs: HealthErr=%q ReadyErr=%q", ns.HealthErr, ns.ReadyErr)
	}
	njs := iomeshclient.FormatConnectionStatusJSON(ns)
	var nparsed map[string]any
	if err := json.Unmarshal([]byte(njs), &nparsed); err != nil {
		t.Fatalf("nil json: %v\n%s", err, njs)
	}
	if nparsed["health_err"] != "nil client" || nparsed["ready_err"] != "nil client" {
		t.Fatalf("nil JSON errs: health_err=%v ready_err=%v\n%s",
			nparsed["health_err"], nparsed["ready_err"], njs)
	}
	nhuman := iomeshclient.FormatConnectionStatus(ns)
	if !strings.Contains(nhuman, "health_err=nil client\n") ||
		!strings.Contains(nhuman, "ready_err=nil client\n") {
		t.Fatalf("nil Format missing probe errs: %q", nhuman)
	}
}

func TestConnectionStatus_AlwaysEmitsIdentity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health", "/ready":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Unset identity: empty strings still present in JSON + Format.
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	s := nc.ConnectionStatus(context.Background())
	if s.Tenant != "" || s.Org != "" || s.Workspace != "" {
		t.Fatalf("unset identity want empty: tenant=%q org=%q workspace=%q",
			s.Tenant, s.Org, s.Workspace)
	}

	js := iomeshclient.FormatConnectionStatusJSON(s)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, js)
	}
	for _, key := range []string{"tenant", "org", "workspace"} {
		v, ok := parsed[key].(string)
		if !ok {
			t.Fatalf("JSON %q missing or not string: %v\n%s", key, parsed[key], js)
		}
		if v != "" {
			t.Fatalf("JSON %q=%q want empty\n%s", key, v, js)
		}
	}

	human := iomeshclient.FormatConnectionStatus(s)
	if !strings.Contains(human, "tenant=\n") || !strings.Contains(human, "org=\n") ||
		!strings.Contains(human, "workspace=\n") {
		t.Fatalf("Format missing empty identity lines: %q", human)
	}

	// Set identity: values appear in JSON + Format.
	nc2, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL},
		iomeshclient.WithTenant("ten"),
		iomeshclient.WithOrg("org1"),
		iomeshclient.WithWorkspace("ws1"),
	)
	if err != nil {
		t.Fatal(err)
	}
	s2 := nc2.ConnectionStatus(context.Background())
	if s2.Tenant != "ten" || s2.Org != "org1" || s2.Workspace != "ws1" {
		t.Fatalf("identity tenant=%q org=%q workspace=%q", s2.Tenant, s2.Org, s2.Workspace)
	}
	js2 := iomeshclient.FormatConnectionStatusJSON(s2)
	var parsed2 map[string]any
	if err := json.Unmarshal([]byte(js2), &parsed2); err != nil {
		t.Fatalf("json2: %v\n%s", err, js2)
	}
	if parsed2["tenant"] != "ten" || parsed2["org"] != "org1" || parsed2["workspace"] != "ws1" {
		t.Fatalf("JSON identity: %v\n%s", parsed2, js2)
	}
	human2 := iomeshclient.FormatConnectionStatus(s2)
	if !strings.Contains(human2, "tenant=ten\n") || !strings.Contains(human2, "org=org1\n") ||
		!strings.Contains(human2, "workspace=ws1\n") {
		t.Fatalf("Format missing set identity: %q", human2)
	}

	// Nil client: empty identity always present in JSON + Format.
	var nilC *iomeshclient.Client
	ns := nilC.ConnectionStatus(context.Background())
	if ns.Tenant != "" || ns.Org != "" || ns.Workspace != "" {
		t.Fatalf("nil identity want empty: %+v", ns)
	}
	njs := iomeshclient.FormatConnectionStatusJSON(ns)
	var nparsed map[string]any
	if err := json.Unmarshal([]byte(njs), &nparsed); err != nil {
		t.Fatalf("nil json: %v\n%s", err, njs)
	}
	for _, key := range []string{"tenant", "org", "workspace"} {
		if _, ok := nparsed[key]; !ok {
			t.Fatalf("nil JSON missing %q: %s", key, njs)
		}
		if nparsed[key] != "" {
			t.Fatalf("nil JSON %q=%v want empty: %s", key, nparsed[key], njs)
		}
	}
	nhuman := iomeshclient.FormatConnectionStatus(ns)
	if !strings.Contains(nhuman, "tenant=\n") || !strings.Contains(nhuman, "org=\n") ||
		!strings.Contains(nhuman, "workspace=\n") {
		t.Fatalf("nil Format missing empty identity: %q", nhuman)
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
