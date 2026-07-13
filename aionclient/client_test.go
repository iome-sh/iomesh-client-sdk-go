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

func newTestClient(t *testing.T) *aionclient.Client {
	t.Helper()

	store := mem.New()
	t.Cleanup(func() { _ = store.Close() })

	b := broker.New(store)
	cm := consumer.NewManager(store, nil)
	hub := ephemeral.New()

	mux := http.NewServeMux()
	httpapi.RegisterRoutes(mux, b, cm, hub, store, nil, nil, "", false, nil)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	nc, err := aionclient.Connect(aionclient.Options{URL: ts.URL})
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	return nc
}

func TestCreateStream(t *testing.T) {
	nc := newTestClient(t)
	ctx := context.Background()

	err := nc.CreateStream(ctx, aionclient.StreamConfig{
		Name:      "AGENT_TASKS",
		Subjects:  []string{"agent.tasks.>"},
		Retention: aionclient.RetentionWorkQueue,
	})
	if err != nil {
		t.Fatalf("CreateStream() error: %v", err)
	}
}

func TestCreateStreamIgnoresConflict(t *testing.T) {
	nc := newTestClient(t)
	ctx := context.Background()

	cfg := aionclient.StreamConfig{
		Name:     "dup",
		Subjects: []string{"dup.>"},
	}
	if err := nc.CreateStream(ctx, cfg); err != nil {
		t.Fatalf("first CreateStream() error: %v", err)
	}
	if err := nc.CreateStream(ctx, cfg); err != nil {
		t.Fatalf("duplicate CreateStream() error: %v, want nil", err)
	}
}

func TestCreateStreamValidation(t *testing.T) {
	nc := newTestClient(t)
	ctx := context.Background()

	if err := nc.CreateStream(ctx, aionclient.StreamConfig{}); err == nil {
		t.Fatal("CreateStream() error = nil, want validation error")
	}
	if err := nc.CreateStream(ctx, aionclient.StreamConfig{Name: "no-subjects"}); err == nil {
		t.Fatal("CreateStream() error = nil, want validation error")
	}
}

func TestEnsureStream(t *testing.T) {
	nc := newTestClient(t)
	ctx := context.Background()

	cfg := aionclient.StreamConfig{
		Name:     "ensured",
		Subjects: []string{"ensured.>"},
	}
	if err := nc.EnsureStream(ctx, cfg); err != nil {
		t.Fatalf("first EnsureStream() error: %v", err)
	}
	if err := nc.EnsureStream(ctx, cfg); err != nil {
		t.Fatalf("second EnsureStream() error: %v", err)
	}
}

func TestPub(t *testing.T) {
	nc := newTestClient(t)
	ctx := context.Background()

	if err := nc.Pub(ctx, "agent.events.worker.started", []byte("started"), map[string]string{
		"trace": "abc",
	}); err != nil {
		t.Fatalf("Pub() error: %v", err)
	}
}

func TestPubInvalidSubject(t *testing.T) {
	nc := newTestClient(t)
	ctx := context.Background()

	err := nc.Pub(ctx, "agent.events.>", []byte("x"), nil)
	if err == nil {
		t.Fatal("Pub() error = nil, want API error")
	}
	var apiErr *aionclient.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Pub() error = %T, want *aionclient.APIError", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusBadRequest)
	}
}

func TestPubValidation(t *testing.T) {
	nc := newTestClient(t)
	ctx := context.Background()

	if err := nc.Pub(ctx, "", []byte("x"), nil); err == nil {
		t.Fatal("Pub() error = nil, want validation error")
	}
}
