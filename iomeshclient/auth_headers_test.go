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
	var gotTenant, gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotTenant = r.Header.Get("X-IOMesh-Tenant")
		gotAuth = r.Header.Get("Authorization")
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	nc, err := iomeshclient.Connect(
		iomeshclient.Options{URL: srv.URL},
		iomeshclient.WithTenant("dept.research"),
		iomeshclient.WithBearerToken("test-token"),
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
