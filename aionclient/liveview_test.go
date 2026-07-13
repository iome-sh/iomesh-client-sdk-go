//go:build ignore

package aionclient_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpapi "github.com/iome-sh/aion/internal/api/http"
	"github.com/iome-sh/aion/internal/broker"
	"github.com/iome-sh/aion/internal/consumer"
	"github.com/iome-sh/aion/internal/ephemeral"
	"github.com/iome-sh/aion/internal/registry"
	"github.com/iome-sh/aion/internal/storage/mem"
	"github.com/iome-sh/aion/pkg/aionclient"
)

func newRegistryTestClient(t *testing.T) *aionclient.Client {
	t.Helper()

	store := mem.New()
	t.Cleanup(func() { _ = store.Close() })

	reg := registry.NewFakeStore()
	b := broker.New(store)
	b.SetProcessorRegistry(reg)
	cm := consumer.NewManager(store, nil)
	hub := ephemeral.New()

	mux := http.NewServeMux()
	httpapi.RegisterRoutes(mux, b, cm, hub, store, reg, nil, "", false, nil)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	nc, err := aionclient.Connect(aionclient.Options{URL: ts.URL})
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	return nc
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func TestLiveViewRegisterProcessor(t *testing.T) {
	nc := newRegistryTestClient(t)
	ctx := context.Background()

	if err := nc.EnsureStream(ctx, aionclient.StreamConfig{
		Name:     "FINANCE_EVENTS",
		Subjects: []string{"dept.finance.events.>"},
	}); err != nil {
		t.Fatalf("EnsureStream() error: %v", err)
	}

	cfg := aionclient.ProcessorConfig{
		ID:           "finance-enricher",
		SourceStream: "FINANCE_EVENTS",
		Tenant:       "dept.finance",
		Type:         aionclient.ProcessorTypeEnrich,
		ConfigJSON:   `{"patch":{"risk_score":0}}`,
	}
	if err := nc.RegisterProcessor(ctx, cfg); err != nil {
		t.Fatalf("RegisterProcessor() error: %v", err)
	}
	if err := nc.RegisterProcessor(ctx, cfg); err != nil {
		t.Fatalf("duplicate RegisterProcessor() error: %v, want nil (409 ignored)", err)
	}
}

func TestLiveViewRegisterProcessorValidation(t *testing.T) {
	nc := newRegistryTestClient(t)
	ctx := context.Background()

	if err := nc.RegisterProcessor(ctx, aionclient.ProcessorConfig{}); err == nil {
		t.Fatal("RegisterProcessor() error = nil, want validation error")
	}
	if err := nc.RegisterProcessor(ctx, aionclient.ProcessorConfig{ID: "only-id"}); err == nil {
		t.Fatal("RegisterProcessor() error = nil, want validation error")
	}
}

func TestLiveViewListLiveViews(t *testing.T) {
	ctx := context.Background()

	store := mem.New()
	t.Cleanup(func() { _ = store.Close() })
	reg := registry.NewFakeStore()
	b := broker.New(store)
	cm := consumer.NewManager(store, nil)
	hub := ephemeral.New()
	mux := http.NewServeMux()
	httpapi.RegisterRoutes(mux, b, cm, hub, store, reg, nil, "", false, nil)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	createBody := map[string]any{
		"id":                   "finance-summary-view",
		"tenant_id":            "finance",
		"name":                 "finance-summary-view",
		"domain":               "finance",
		"owner":                "finance-team",
		"upstream_product_ids": []string{"finance-raw-events"},
		"subjects":             []string{"dept.finance.events.>"},
		"processor_ids":        []string{"finance-enricher"},
		"freshness_slo_sec":    300,
	}
	resp := postJSON(t, server.URL+"/v3/registry/liveviews", createBody)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create live view status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	_ = resp.Body.Close()

	listClient, err := aionclient.Connect(aionclient.Options{URL: server.URL})
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}

	views, err := listClient.ListLiveViews(ctx, "finance")
	if err != nil {
		t.Fatalf("ListLiveViews() error: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("ListLiveViews() len = %d, want 1", len(views))
	}
	if views[0].ID != "finance-summary-view" {
		t.Fatalf("ListLiveViews()[0].ID = %q, want finance-summary-view", views[0].ID)
	}
	if views[0].TenantID != "finance" {
		t.Fatalf("ListLiveViews()[0].TenantID = %q, want finance", views[0].TenantID)
	}
	if len(views[0].ProcessorIDs) != 1 || views[0].ProcessorIDs[0] != "finance-enricher" {
		t.Fatalf("ListLiveViews()[0].ProcessorIDs = %v, want [finance-enricher]", views[0].ProcessorIDs)
	}
}

func TestLiveViewListRequiresTenant(t *testing.T) {
	nc := newRegistryTestClient(t)
	ctx := context.Background()

	if _, err := nc.ListLiveViews(ctx, ""); err == nil {
		t.Fatal("ListLiveViews() error = nil, want validation error")
	}
}

func newRegistryTestClientWithTenant(t *testing.T, tenant string) *aionclient.Client {
	t.Helper()

	store := mem.New()
	t.Cleanup(func() { _ = store.Close() })

	reg := registry.NewFakeStore()
	b := broker.New(store)
	b.SetProcessorRegistry(reg)
	cm := consumer.NewManager(store, nil)
	hub := ephemeral.New()

	mux := http.NewServeMux()
	httpapi.RegisterRoutes(mux, b, cm, hub, store, reg, nil, "", false, nil)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	opts := []aionclient.ConnectOpt{}
	if tenant != "" {
		opts = append(opts, aionclient.WithTenant(tenant))
	}
	nc, err := aionclient.Connect(aionclient.Options{URL: ts.URL}, opts...)
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	return nc
}

func TestLiveViewEnrichmentPublishRoundTrip(t *testing.T) {
	nc := newRegistryTestClientWithTenant(t, "dept.finance")
	ctx := context.Background()

	if err := nc.EnsureStream(ctx, aionclient.StreamConfig{
		Name:     "FINANCE_EVENTS",
		Subjects: []string{"dept.finance.events.>"},
	}); err != nil {
		t.Fatalf("EnsureStream() error: %v", err)
	}

	if err := nc.RegisterProcessor(ctx, aionclient.ProcessorConfig{
		ID:           "finance-enricher",
		SourceStream: "FINANCE_EVENTS",
		Tenant:       "dept.finance",
		Type:         aionclient.ProcessorTypeEnrich,
		ConfigJSON:   `{"patch":{"risk_score":0}}`,
	}); err != nil {
		t.Fatalf("RegisterProcessor() error: %v", err)
	}

	ack, err := nc.Publish(ctx, "FINANCE_EVENTS", "dept.finance.events.trade", []byte(`{"amount":100}`))
	if err != nil {
		t.Fatalf("Publish() error: %v", err)
	}
	if ack.Seq == 0 {
		t.Fatal("Publish() seq = 0, want > 0")
	}

	sub, err := nc.PullSubscribe(ctx, aionclient.PullSubscribeConfig{
		Stream:   "FINANCE_EVENTS",
		Consumer: "enrichment-pilot-test",
		Filter:   "dept.finance.events.>",
	})
	if err != nil {
		t.Fatalf("PullSubscribe() error: %v", err)
	}

	msgs, err := sub.Fetch(1, aionclient.MaxWait(5*time.Second))
	if err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("Fetch() len = %d, want 1", len(msgs))
	}

	var payload map[string]any
	if err := json.Unmarshal(msgs[0].Data(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["amount"] != float64(100) {
		t.Fatalf("amount = %v, want 100", payload["amount"])
	}
	if payload["risk_score"] != float64(0) {
		t.Fatalf("risk_score = %v, want 0 (enriched)", payload["risk_score"])
	}
}
