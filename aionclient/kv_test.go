//go:build ignore

package aionclient_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	httpapi "github.com/iome-sh/aion/internal/api/http"
	"github.com/iome-sh/aion/internal/broker"
	"github.com/iome-sh/aion/internal/consumer"
	"github.com/iome-sh/aion/internal/ephemeral"
	"github.com/iome-sh/aion/internal/storage/mem"
	"github.com/iome-sh/aion/pkg/aionclient"
)

func newKVTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	store := mem.New()
	t.Cleanup(func() { _ = store.Close() })

	b := broker.New(store)
	cm := consumer.NewManager(store, nil)
	hub := ephemeral.New()

	mux := http.NewServeMux()
	httpapi.RegisterRoutes(mux, b, cm, hub, store, nil, nil, "", false, nil)
	return httptest.NewServer(mux)
}

func TestKVPutGetDeleteList(t *testing.T) {
	ts := newKVTestServer(t)
	defer ts.Close()

	ctx := context.Background()
	nc, err := aionclient.Connect(aionclient.Options{URL: ts.URL})
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}

	if err := nc.CreateBucket(ctx, "agent-state"); err != nil {
		t.Fatalf("CreateBucket() error: %v", err)
	}

	rev, err := nc.Put(ctx, "agent-state", "worker-1.checkpoint", []byte("seq=42"))
	if err != nil {
		t.Fatalf("Put() error: %v", err)
	}
	if rev != 1 {
		t.Fatalf("revision = %d, want 1", rev)
	}

	entry, err := nc.Get(ctx, "agent-state", "worker-1.checkpoint")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if string(entry.Value) != "seq=42" {
		t.Fatalf("value = %q, want seq=42", entry.Value)
	}
	if entry.Revision != 1 {
		t.Fatalf("revision = %d, want 1", entry.Revision)
	}

	rev, err = nc.Put(ctx, "agent-state", "worker-2.checkpoint", []byte("seq=99"))
	if err != nil {
		t.Fatalf("Put(worker-2) error: %v", err)
	}
	if rev != 2 {
		t.Fatalf("worker-2 revision = %d, want 2", rev)
	}

	keys, err := nc.ListKeys(ctx, "agent-state", "worker-")
	if err != nil {
		t.Fatalf("ListKeys() error: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("keys = %v, want 2 entries", keys)
	}

	if err := nc.Delete(ctx, "agent-state", "worker-1.checkpoint"); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err = nc.Get(ctx, "agent-state", "worker-1.checkpoint")
	if err == nil {
		t.Fatal("Get() after delete error = nil, want not found")
	}
	var apiErr *aionclient.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("Get() after delete error = %v, want HTTP 404", err)
	}
}
