package iomeshclient_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
)

func TestRegisterMemoryProductAndPublishIngest(t *testing.T) {
	var mu sync.Mutex
	var created []iomeshclient.MemoryProductConfig
	var published []map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v5/registry/memory-products", func(w http.ResponseWriter, r *http.Request) {
		var cfg iomeshclient.MemoryProductConfig
		_ = json.NewDecoder(r.Body).Decode(&cfg)
		mu.Lock()
		created = append(created, cfg)
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(cfg)
	})
	mux.HandleFunc("POST /v1/streams/MEMORY_INGEST/publish", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		published = append(published, body)
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"stream":  "MEMORY_INGEST",
			"seq":     1,
			"subject": body["subject"],
		})
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: ts.URL}, iomeshclient.WithTenant("dept.research"))
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}

	ctx := context.Background()
	if err := nc.RegisterMemoryProduct(ctx, iomeshclient.MemoryProductConfig{
		ProductID:  "research.memory",
		TenantID:   "dept.research",
		PalaceRoot: "iomesh-memory/dept.research/palace",
	}); err != nil {
		t.Fatalf("RegisterMemoryProduct: %v", err)
	}

	if _, err := nc.PublishMemoryIngest(ctx, "dept.research", iomeshclient.MemoryEnvelope{
		Role:    "assistant",
		Content: "sdk ingest smoke",
	}); err != nil {
		t.Fatalf("PublishMemoryIngest: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(created) != 1 {
		t.Fatalf("created = %d, want 1", len(created))
	}
	if len(published) != 1 {
		t.Fatalf("published = %d, want 1", len(published))
	}
}

func TestPublishMemoryIngestMarshalsTemporalFields(t *testing.T) {
	var mu sync.Mutex
	var rawPayload []byte

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/streams/MEMORY_INGEST/publish", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if s, ok := body["payload"].(string); ok {
			decoded, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				t.Errorf("payload base64: %v", err)
			} else {
				mu.Lock()
				rawPayload = decoded
				mu.Unlock()
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"stream":  "MEMORY_INGEST",
			"seq":     7,
			"subject": body["subject"],
		})
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: ts.URL}, iomeshclient.WithTenant("dept.research"))
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}

	env := iomeshclient.MemoryEnvelope{
		Role:           "user",
		Content:        "fix rotation due Q3",
		SessionID:      "sess-1",
		TurnID:         "turn-1",
		EventTime:      "2026-07-13T12:00:00Z",
		SessionSeq:     3,
		SourceStream:   "MEMORY_INGEST",
		SourceSeq:      42,
		CausalParentID: "mem-parent",
		EntityRefs: []iomeshclient.MemoryEntityRef{
			{Type: "ticket", ID: "JIRA-100"},
		},
		ValidFrom:  "2026-07-01T00:00:00Z",
		ValidUntil: "2026-12-31T23:59:59Z",
	}
	if _, err := nc.PublishMemoryIngest(context.Background(), "dept.research", env); err != nil {
		t.Fatalf("PublishMemoryIngest: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(rawPayload) == 0 {
		t.Fatal("expected published payload")
	}

	var got map[string]any
	if err := json.Unmarshal(rawPayload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if got["type"] != "memory_ingest" {
		t.Fatalf("type = %v, want memory_ingest", got["type"])
	}
	if got["event_time"] != "2026-07-13T12:00:00Z" {
		t.Fatalf("event_time = %v", got["event_time"])
	}
	if got["session_seq"] != float64(3) {
		t.Fatalf("session_seq = %v", got["session_seq"])
	}
	if got["source_stream"] != "MEMORY_INGEST" {
		t.Fatalf("source_stream = %v", got["source_stream"])
	}
	if got["source_seq"] != float64(42) {
		t.Fatalf("source_seq = %v", got["source_seq"])
	}
	if got["causal_parent_id"] != "mem-parent" {
		t.Fatalf("causal_parent_id = %v", got["causal_parent_id"])
	}
	if got["valid_from"] != "2026-07-01T00:00:00Z" {
		t.Fatalf("valid_from = %v", got["valid_from"])
	}
	if got["valid_until"] != "2026-12-31T23:59:59Z" {
		t.Fatalf("valid_until = %v", got["valid_until"])
	}
	refs, ok := got["entity_refs"].([]any)
	if !ok || len(refs) != 1 {
		t.Fatalf("entity_refs = %v", got["entity_refs"])
	}
	ref0, _ := refs[0].(map[string]any)
	if ref0["type"] != "ticket" || ref0["id"] != "JIRA-100" {
		t.Fatalf("entity_refs[0] = %v", ref0)
	}
}

func TestMemoryEnvelopeTemporalOmitEmpty(t *testing.T) {
	b, err := json.Marshal(iomeshclient.MemoryEnvelope{
		Type:    "memory_ingest",
		Content: "minimal",
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(b)
	for _, key := range []string{
		"event_time", "ingested_at", "source_stream", "source_seq",
		"session_seq", "causal_parent_id", "entity_refs", "valid_from", "valid_until",
	} {
		if strings.Contains(s, key) {
			t.Fatalf("expected omitempty to drop %q from %s", key, s)
		}
	}
}

func TestRetrieveMemorySuccess(t *testing.T) {
	var mu sync.Mutex
	var gotBody map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v5/memory/retrieve", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		gotBody = body
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"memories": []map[string]any{
				{
					"id":          "mem-1",
					"summary":     "lease rotation",
					"full":        "lease rotation due Q3",
					"score":       0.91,
					"confidence":  0.8,
					"timestamp":   "2026-07-13T12:00:00Z",
					"turn_id":     "turn-1",
					"event_time":  "2026-07-13T12:00:00Z",
					"session_seq": 3,
				},
			},
		})
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: ts.URL}, iomeshclient.WithTenant("dept.research"))
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}

	resp, err := nc.RetrieveMemory(context.Background(), iomeshclient.MemoryRetrieveRequest{
		TenantID:   "dept.research",
		Query:      "lease rotation",
		Limit:      5,
		SessionID:  "sess-1",
		SessionSeq: 3,
		Since:      "2026-07-01T00:00:00Z",
		Until:      "2026-07-31T23:59:59Z",
	})
	if err != nil {
		t.Fatalf("RetrieveMemory: %v", err)
	}
	if len(resp.Memories) != 1 {
		t.Fatalf("memories = %d, want 1", len(resp.Memories))
	}
	hit := resp.Memories[0]
	if hit.ID != "mem-1" || hit.Summary != "lease rotation" || hit.Full != "lease rotation due Q3" {
		t.Fatalf("hit = %+v", hit)
	}
	if hit.Score != 0.91 || hit.SessionSeq != 3 {
		t.Fatalf("score/session_seq = %+v", hit)
	}

	mu.Lock()
	defer mu.Unlock()
	if gotBody["tenant_id"] != "dept.research" {
		t.Fatalf("tenant_id = %v", gotBody["tenant_id"])
	}
	if gotBody["query"] != "lease rotation" {
		t.Fatalf("query = %v", gotBody["query"])
	}
	if gotBody["type"] != "memory_recall" {
		t.Fatalf("type = %v", gotBody["type"])
	}
	if gotBody["limit"] != float64(5) {
		t.Fatalf("limit = %v", gotBody["limit"])
	}
	if gotBody["session_id"] != "sess-1" {
		t.Fatalf("session_id = %v", gotBody["session_id"])
	}
	if gotBody["session_seq"] != float64(3) {
		t.Fatalf("session_seq = %v", gotBody["session_seq"])
	}
	if gotBody["since"] != "2026-07-01T00:00:00Z" {
		t.Fatalf("since = %v", gotBody["since"])
	}
	if gotBody["until"] != "2026-07-31T23:59:59Z" {
		t.Fatalf("until = %v", gotBody["until"])
	}
}

func TestRetrieveMemoryEmptyHits(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v5/memory/retrieve", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"memories": nil})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: ts.URL})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	resp, err := nc.RetrieveMemory(context.Background(), iomeshclient.MemoryRetrieveRequest{
		TenantID: "dept.research",
		Query:    "nothing",
	})
	if err != nil {
		t.Fatalf("RetrieveMemory: %v", err)
	}
	if resp.Memories == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(resp.Memories) != 0 {
		t.Fatalf("len = %d", len(resp.Memories))
	}
}

func TestRetrieveMemoryValidation(t *testing.T) {
	ts := httptest.NewServer(http.NewServeMux())
	defer ts.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: ts.URL})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	ctx := context.Background()

	if _, err := nc.RetrieveMemory(ctx, iomeshclient.MemoryRetrieveRequest{Query: "x"}); err == nil {
		t.Fatal("expected tenant_id required")
	} else if !strings.Contains(err.Error(), "tenant_id") {
		t.Fatalf("err = %v, want tenant_id", err)
	}

	if _, err := nc.RetrieveMemory(ctx, iomeshclient.MemoryRetrieveRequest{TenantID: "dept.research"}); err == nil {
		t.Fatal("expected query or session_id required")
	} else if !strings.Contains(err.Error(), "query or session_id") {
		t.Fatalf("err = %v, want query or session_id", err)
	}

	// Whitespace-only also rejected.
	if _, err := nc.RetrieveMemory(ctx, iomeshclient.MemoryRetrieveRequest{
		TenantID: "  ",
		Query:    "ok",
	}); err == nil {
		t.Fatal("expected tenant_id required for whitespace")
	}
	if _, err := nc.RetrieveMemory(ctx, iomeshclient.MemoryRetrieveRequest{
		TenantID: "dept.research",
		Query:    "   ",
	}); err == nil {
		t.Fatal("expected query or session_id required for whitespace")
	}
}

func TestRetrieveMemoryV1ThenV5Fallback(t *testing.T) {
	var paths []string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/memory/retrieve", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("POST /v5/memory/retrieve", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{"memories": []any{}})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: ts.URL})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	resp, err := nc.RetrieveMemory(context.Background(), iomeshclient.MemoryRetrieveRequest{
		TenantID:  "dept.x",
		SessionID: "sess-only", // query may be empty when session_id set
	})
	if err != nil {
		t.Fatalf("RetrieveMemory: %v", err)
	}
	if resp.Path != "/v5/memory/retrieve" {
		t.Fatalf("path=%q", resp.Path)
	}
	if len(paths) != 2 || paths[0] != "/v1/memory/retrieve" || paths[1] != "/v5/memory/retrieve" {
		t.Fatalf("paths=%v", paths)
	}
}

func TestRequestMemoryRecallFullSessionID(t *testing.T) {
	var mu sync.Mutex
	var rawPayload []byte
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/streams/MEMORY_RPC/publish", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if s, ok := body["payload"].(string); ok {
			decoded, _ := base64.StdEncoding.DecodeString(s)
			mu.Lock()
			rawPayload = decoded
			mu.Unlock()
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"stream": "MEMORY_RPC", "seq": 2, "subject": body["subject"]})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: ts.URL})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	ack, err := nc.RequestMemoryRecallFull(context.Background(), iomeshclient.MemoryRecallRequest{
		TenantID:  "dept.research",
		Query:     "find notes",
		Limit:     8,
		SessionID: "dept.research.mesh-dogfood",
	})
	if err != nil {
		t.Fatalf("RequestMemoryRecallFull: %v", err)
	}
	if ack == nil || ack.Seq != 2 {
		t.Fatalf("ack=%+v", ack)
	}
	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(rawPayload, &got); err != nil {
		t.Fatal(err)
	}
	if got["session_id"] != "dept.research.mesh-dogfood" {
		t.Fatalf("session_id=%v", got["session_id"])
	}
	if got["query"] != "find notes" {
		t.Fatalf("query=%v", got["query"])
	}
}

func TestIngestMemoryTurnSuccess(t *testing.T) {
	var mu sync.Mutex
	var gotBody map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v5/memory/ingest", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		gotBody = body
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":    "ok",
			"memory_id": "mem-99",
			"tier":      1,
			"ingested":  1,
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: ts.URL})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}

	resp, err := nc.IngestMemoryTurn(context.Background(), "dept.research", iomeshclient.MemoryEnvelope{
		Role:       "assistant",
		Content:    "noted lease rotation",
		SessionID:  "sess-1",
		EventTime:  "2026-07-13T15:00:00Z",
		SessionSeq: 4,
	})
	if err != nil {
		t.Fatalf("IngestMemoryTurn: %v", err)
	}
	if resp.Status != "ok" || resp.MemoryID != "mem-99" || resp.Ingested != 1 {
		t.Fatalf("resp = %+v", resp)
	}

	mu.Lock()
	defer mu.Unlock()
	if gotBody["tenant_id"] != "dept.research" {
		t.Fatalf("tenant_id = %v", gotBody["tenant_id"])
	}
	if gotBody["type"] != "memory_ingest" {
		t.Fatalf("type = %v", gotBody["type"])
	}
	if gotBody["content"] != "noted lease rotation" {
		t.Fatalf("content = %v", gotBody["content"])
	}
	if gotBody["event_time"] != "2026-07-13T15:00:00Z" {
		t.Fatalf("event_time = %v", gotBody["event_time"])
	}
	if gotBody["session_seq"] != float64(4) {
		t.Fatalf("session_seq = %v", gotBody["session_seq"])
	}
}

func TestIngestMemoryTurnValidation(t *testing.T) {
	ts := httptest.NewServer(http.NewServeMux())
	defer ts.Close()
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: ts.URL})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	ctx := context.Background()
	if _, err := nc.IngestMemoryTurn(ctx, "", iomeshclient.MemoryEnvelope{Content: "x"}); err == nil {
		t.Fatal("expected tenant_id required")
	}
	if _, err := nc.IngestMemoryTurn(ctx, "dept.research", iomeshclient.MemoryEnvelope{}); err == nil {
		t.Fatal("expected content required")
	}
}
