package iomeshclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
)

func TestConnectSetsTenantAndBearerHeaders(t *testing.T) {
	var mu sync.Mutex
	var gotTenant, gotAuth, gotOrg, gotWS, gotUA string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotTenant = r.Header.Get("X-IOMesh-Tenant")
		gotAuth = r.Header.Get("Authorization")
		gotOrg = r.Header.Get("X-IOMesh-Org")
		gotWS = r.Header.Get("X-IOMesh-Workspace")
		gotUA = r.Header.Get("User-Agent")
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	nc, err := iomeshclient.Connect(
		iomeshclient.Options{URL: srv.URL},
		iomeshclient.WithTenant("dept.research"),
		iomeshclient.WithBearerToken("test-token"),
		iomeshclient.WithOrg("org_a"),
		iomeshclient.WithWorkspace("ws_1"),
	)
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}

	if err := nc.Pub(context.Background(), "dept.research.events", []byte("x"), nil); err != nil {
		t.Fatalf("Pub() error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if gotTenant != "dept.research" {
		t.Fatalf("X-IOMesh-Tenant = %q, want dept.research", gotTenant)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("Authorization = %q, want Bearer test-token", gotAuth)
	}
	if gotOrg != "org_a" {
		t.Fatalf("X-IOMesh-Org = %q, want org_a", gotOrg)
	}
	if gotWS != "ws_1" {
		t.Fatalf("X-IOMesh-Workspace = %q, want ws_1", gotWS)
	}
	wantUA := "iomesh-client-sdk-go/" + iomeshclient.Version
	if gotUA != wantUA {
		t.Fatalf("User-Agent = %q, want %q", gotUA, wantUA)
	}
}

func TestWithUserAgentOverride(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	nc, err := iomeshclient.Connect(
		iomeshclient.Options{URL: srv.URL},
		iomeshclient.WithUserAgent("my-agent/1.0"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := nc.Pub(context.Background(), "events", []byte("x"), nil); err != nil {
		t.Fatal(err)
	}
	if gotUA != "my-agent/1.0" {
		t.Fatalf("User-Agent = %q", gotUA)
	}
}

func TestConnectRejectsUnsafeURLs(t *testing.T) {
	cases := []string{
		"",
		"file:///etc/passwd",
		"ftp://example.com",
		"//no-scheme.example",
		"http://user:pass@127.0.0.1:8422",
		"not a url",
	}
	for _, u := range cases {
		_, err := iomeshclient.Connect(iomeshclient.Options{URL: u})
		if err == nil {
			t.Fatalf("Connect(%q) expected error", u)
		}
	}
	// Valid schemes
	for _, u := range []string{"http://127.0.0.1:8422", "https://mesh.example.com"} {
		if _, err := iomeshclient.Connect(iomeshclient.Options{URL: u}); err != nil {
			t.Fatalf("Connect(%q) unexpected error: %v", u, err)
		}
	}
}

func TestConnectOmitsHeadersWhenUnset(t *testing.T) {
	var mu sync.Mutex
	var gotTenant, gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotTenant = r.Header.Get("X-IOMesh-Tenant")
		gotAuth = r.Header.Get("Authorization")
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}

	if err := nc.Pub(context.Background(), "events.demo", []byte("x"), nil); err != nil {
		t.Fatalf("Pub() error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if gotTenant != "" {
		t.Fatalf("X-IOMesh-Tenant = %q, want empty", gotTenant)
	}
	if gotAuth != "" {
		t.Fatalf("Authorization = %q, want empty", gotAuth)
	}
}
