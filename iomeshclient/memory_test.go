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

func TestRetrieveMemoryRelatedSuccess(t *testing.T) {
	var mu sync.Mutex
	var gotBody map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v5/memory/related", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		gotBody = body
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"memories": []map[string]any{
				{
					"id":           "mem-seed",
					"summary":      "alice note",
					"full":         "person:alice owns widget",
					"score":        0.95,
					"hop_distance": 0,
				},
				{
					"id":           "mem-hop2",
					"summary":      "widget note",
					"full":         "widget depends on org:acme",
					"score":        0.7,
					"hop_distance": 2,
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

	resp, err := nc.RetrieveMemoryRelated(context.Background(), iomeshclient.MemoryRelatedRequest{
		TenantID:   "dept.research",
		SeedEntity: "person:alice",
		Query:      "widget ownership",
		MaxHops:    2,
		Limit:      10,
		SessionID:  "sess-1",
		AsOf:       "2026-08-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("RetrieveMemoryRelated: %v", err)
	}
	if len(resp.Memories) != 2 {
		t.Fatalf("memories = %d, want 2", len(resp.Memories))
	}
	if resp.Memories[0].HopDistance != 0 {
		t.Fatalf("hop_distance[0] = %d, want 0", resp.Memories[0].HopDistance)
	}
	if resp.Memories[1].HopDistance != 2 {
		t.Fatalf("hop_distance[1] = %d, want 2", resp.Memories[1].HopDistance)
	}
	if resp.Memories[1].ID != "mem-hop2" || resp.Memories[1].Summary != "widget note" {
		t.Fatalf("hit[1] = %+v", resp.Memories[1])
	}

	mu.Lock()
	defer mu.Unlock()
	if gotBody["tenant_id"] != "dept.research" {
		t.Fatalf("tenant_id = %v", gotBody["tenant_id"])
	}
	if gotBody["seed_entity"] != "person:alice" {
		t.Fatalf("seed_entity = %v", gotBody["seed_entity"])
	}
	if gotBody["query"] != "widget ownership" {
		t.Fatalf("query = %v", gotBody["query"])
	}
	if gotBody["max_hops"] != float64(2) {
		t.Fatalf("max_hops = %v", gotBody["max_hops"])
	}
	if gotBody["limit"] != float64(10) {
		t.Fatalf("limit = %v", gotBody["limit"])
	}
	if gotBody["session_id"] != "sess-1" {
		t.Fatalf("session_id = %v", gotBody["session_id"])
	}
	if gotBody["as_of"] != "2026-08-01T00:00:00Z" {
		t.Fatalf("as_of = %v", gotBody["as_of"])
	}
	// nil PreferShorterHops: omit key so kernel default true applies (s1286 / aion s1277).
	if _, ok := gotBody["prefer_shorter_hops"]; ok {
		t.Fatalf("prefer_shorter_hops present when nil: %v", gotBody["prefer_shorter_hops"])
	}
}

// TestRetrieveMemoryRelatedPreferShorterHops covers s1286 body wiring:
// false and true appear in the request body; nil omits the key (kernel default true).
// Honesty: multi-hop lite · not full graph RAG · not Memory GA · dual_write OFF.
func TestRetrieveMemoryRelatedPreferShorterHops(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }

	cases := []struct {
		name    string
		pref    *bool
		wantKey bool
		wantVal any
	}{
		{name: "false", pref: boolPtr(false), wantKey: true, wantVal: false},
		{name: "true", pref: boolPtr(true), wantKey: true, wantVal: true},
		{name: "nil_omit", pref: nil, wantKey: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var gotBody map[string]any

			mux := http.NewServeMux()
			mux.HandleFunc("POST /v5/memory/related", func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				mu.Lock()
				gotBody = body
				mu.Unlock()
				_ = json.NewEncoder(w).Encode(map[string]any{
					"memories": []map[string]any{
						{"id": "m1", "summary": "related", "hop_distance": 1},
					},
				})
			})
			// v1 404 so path cascade lands on v5 (same as success test).
			mux.HandleFunc("POST /v1/memory/related", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			})

			ts := httptest.NewServer(mux)
			defer ts.Close()

			nc, err := iomeshclient.Connect(iomeshclient.Options{URL: ts.URL})
			if err != nil {
				t.Fatalf("Connect: %v", err)
			}

			_, err = nc.RetrieveMemoryRelated(context.Background(), iomeshclient.MemoryRelatedRequest{
				TenantID:          "dept.research",
				SeedEntity:        "person:alice",
				PreferShorterHops: tc.pref,
			})
			if err != nil {
				t.Fatalf("RetrieveMemoryRelated: %v", err)
			}

			mu.Lock()
			defer mu.Unlock()
			val, ok := gotBody["prefer_shorter_hops"]
			if ok != tc.wantKey {
				t.Fatalf("prefer_shorter_hops present=%v want=%v body=%v", ok, tc.wantKey, gotBody)
			}
			if tc.wantKey && val != tc.wantVal {
				t.Fatalf("prefer_shorter_hops = %v (%T), want %v", val, val, tc.wantVal)
			}
		})
	}
}

func TestRetrieveMemoryRelatedV1ThenV5Fallback(t *testing.T) {
	var paths []string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/memory/related", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("POST /v5/memory/related", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"memories": []map[string]any{
				{"id": "m1", "summary": "related", "hop_distance": 1},
			},
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: ts.URL})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	resp, err := nc.RetrieveMemoryRelated(context.Background(), iomeshclient.MemoryRelatedRequest{
		TenantID:   "dept.x",
		SeedEntity: "person:alice",
	})
	if err != nil {
		t.Fatalf("RetrieveMemoryRelated: %v", err)
	}
	if resp.Path != "/v5/memory/related" {
		t.Fatalf("path=%q", resp.Path)
	}
	if len(paths) != 2 || paths[0] != "/v1/memory/related" || paths[1] != "/v5/memory/related" {
		t.Fatalf("paths=%v", paths)
	}
	if len(resp.Memories) != 1 || resp.Memories[0].HopDistance != 1 {
		t.Fatalf("memories=%+v", resp.Memories)
	}
}

func TestRetrieveMemoryRelatedValidation(t *testing.T) {
	ts := httptest.NewServer(http.NewServeMux())
	defer ts.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: ts.URL})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	ctx := context.Background()

	if _, err := nc.RetrieveMemoryRelated(ctx, iomeshclient.MemoryRelatedRequest{SeedEntity: "person:alice"}); err == nil {
		t.Fatal("expected tenant_id required")
	} else if !strings.Contains(err.Error(), "tenant_id") {
		t.Fatalf("err = %v, want tenant_id", err)
	}

	if _, err := nc.RetrieveMemoryRelated(ctx, iomeshclient.MemoryRelatedRequest{TenantID: "dept.research"}); err == nil {
		t.Fatal("expected seed_entity or query required")
	} else if !strings.Contains(err.Error(), "seed_entity or query") {
		t.Fatalf("err = %v, want seed_entity or query", err)
	}

	// Whitespace-only also rejected.
	if _, err := nc.RetrieveMemoryRelated(ctx, iomeshclient.MemoryRelatedRequest{
		TenantID:   "  ",
		SeedEntity: "person:alice",
	}); err == nil {
		t.Fatal("expected tenant_id required for whitespace")
	}
	if _, err := nc.RetrieveMemoryRelated(ctx, iomeshclient.MemoryRelatedRequest{
		TenantID:   "dept.research",
		SeedEntity: "   ",
		Query:      "  ",
	}); err == nil {
		t.Fatal("expected seed_entity or query required for whitespace")
	}
}

func TestExportOpsDigestSuccess(t *testing.T) {
	var mu sync.Mutex
	var gotBody map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v5/memory/ops_digest", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		gotBody = body
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"window":  "day",
			"horizon": "ops",
			"as_of":   "2026-08-04T12:00:00Z",
			"since":   "2026-08-03T12:00:00Z",
			"honesty": map[string]any{
				"ops_pulse":          "ga_path",
				"knowledge":          "beta",
				"analytical":         "beta",
				"never_invent_ga":    true,
				"dual_write_default": "off",
				"book_demo":          "off",
				"note":               "Ops digests synthesize live pulse",
			},
			"patterns": []map[string]any{
				{
					"id":      "pat-1",
					"kind":    "recurrence",
					"subject": "setup-confusion",
					"count":   3,
					"window":  "day",
					"score":   0.8,
					"summary": "setup confusion recurring",
				},
			},
			"receipts": []map[string]any{
				{
					"id":          "mem-r1",
					"event_time":  "2026-08-04T10:00:00Z",
					"summary":     "incident setup note",
					"source_hint": "palace_timeline",
				},
			},
			"decision_stub": map[string]any{
				"pattern":      "setup-confusion",
				"receipts_ref": []string{"mem-r1"},
			},
		})
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: ts.URL}, iomeshclient.WithTenant("dept.research"))
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}

	resp, err := nc.ExportOpsDigest(context.Background(), iomeshclient.MemoryOpsDigestRequest{
		TenantID: "dept.research",
		Window:   "day",
		Horizon:  "ops",
		Limit:    10,
		AsOf:     "2026-08-04T12:00:00Z",
	})
	if err != nil {
		t.Fatalf("ExportOpsDigest: %v", err)
	}
	if resp.Path != "/v5/memory/ops_digest" {
		t.Fatalf("path=%q", resp.Path)
	}
	if resp.Window != "day" || resp.Horizon != "ops" {
		t.Fatalf("window/horizon = %s/%s", resp.Window, resp.Horizon)
	}
	if resp.AsOf != "2026-08-04T12:00:00Z" || resp.Since != "2026-08-03T12:00:00Z" {
		t.Fatalf("as_of/since = %s/%s", resp.AsOf, resp.Since)
	}
	if !resp.Honesty.NeverInventGA || resp.Honesty.OpsPulse != "ga_path" || resp.Honesty.Knowledge != "beta" {
		t.Fatalf("honesty unexpected: %+v", resp.Honesty)
	}
	if resp.Honesty.DualWriteDefault != "off" || resp.Honesty.BookDemo != "off" {
		t.Fatalf("dual_write/book_demo honesty: %+v", resp.Honesty)
	}
	if len(resp.Patterns) != 1 || resp.Patterns[0].Subject != "setup-confusion" {
		t.Fatalf("patterns=%+v", resp.Patterns)
	}
	if len(resp.Receipts) != 1 || resp.Receipts[0].ID != "mem-r1" {
		t.Fatalf("receipts=%+v", resp.Receipts)
	}
	if resp.DecisionStub.Pattern != "setup-confusion" || len(resp.DecisionStub.ReceiptsRef) != 1 {
		t.Fatalf("decision_stub=%+v", resp.DecisionStub)
	}

	mu.Lock()
	defer mu.Unlock()
	if gotBody["tenant_id"] != "dept.research" {
		t.Fatalf("tenant_id = %v", gotBody["tenant_id"])
	}
	if gotBody["window"] != "day" {
		t.Fatalf("window = %v", gotBody["window"])
	}
	if gotBody["horizon"] != "ops" {
		t.Fatalf("horizon = %v", gotBody["horizon"])
	}
	if gotBody["limit"] != float64(10) {
		t.Fatalf("limit = %v", gotBody["limit"])
	}
	if gotBody["as_of"] != "2026-08-04T12:00:00Z" {
		t.Fatalf("as_of = %v", gotBody["as_of"])
	}
}

func TestExportOpsDigestV1ThenV5Fallback(t *testing.T) {
	var paths []string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/memory/ops_digest", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("POST /v5/memory/ops_digest", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"window":  "week",
			"horizon": "all",
			"as_of":   "2026-08-04T00:00:00Z",
			"honesty": map[string]any{
				"ops_pulse":          "ga_path",
				"knowledge":          "beta",
				"analytical":         "beta",
				"never_invent_ga":    true,
				"dual_write_default": "off",
				"book_demo":          "off",
			},
			"patterns": []map[string]any{},
			"receipts": []map[string]any{
				{"id": "r1", "summary": "week receipt"},
			},
			"decision_stub": map[string]any{},
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: ts.URL})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	resp, err := nc.ExportOpsDigest(context.Background(), iomeshclient.MemoryOpsDigestRequest{
		TenantID: "dept.x",
		Window:   "week",
		Horizon:  "all",
	})
	if err != nil {
		t.Fatalf("ExportOpsDigest: %v", err)
	}
	if resp.Path != "/v5/memory/ops_digest" {
		t.Fatalf("path=%q", resp.Path)
	}
	if len(paths) != 2 || paths[0] != "/v1/memory/ops_digest" || paths[1] != "/v5/memory/ops_digest" {
		t.Fatalf("paths=%v", paths)
	}
	if resp.Window != "week" || resp.Horizon != "all" {
		t.Fatalf("window/horizon = %s/%s", resp.Window, resp.Horizon)
	}
	if len(resp.Receipts) != 1 || resp.Receipts[0].ID != "r1" {
		t.Fatalf("receipts=%+v", resp.Receipts)
	}
	if resp.Patterns == nil {
		t.Fatal("patterns should be non-nil empty slice")
	}
}

func TestExportOpsDigestDefaultsAndValidation(t *testing.T) {
	var mu sync.Mutex
	var gotBody map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v5/memory/ops_digest", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		gotBody = body
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"window":  body["window"],
			"horizon": body["horizon"],
			"as_of":   "2026-08-04T00:00:00Z",
			"honesty": map[string]any{
				"ops_pulse":          "ga_path",
				"knowledge":          "beta",
				"analytical":         "beta",
				"never_invent_ga":    true,
				"dual_write_default": "off",
				"book_demo":          "off",
			},
			"patterns":      []any{},
			"receipts":      []any{},
			"decision_stub": map[string]any{},
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: ts.URL})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	ctx := context.Background()

	if _, err := nc.ExportOpsDigest(ctx, iomeshclient.MemoryOpsDigestRequest{}); err == nil {
		t.Fatal("expected tenant_id required")
	} else if !strings.Contains(err.Error(), "tenant_id") {
		t.Fatalf("err = %v, want tenant_id", err)
	}
	if _, err := nc.ExportOpsDigest(ctx, iomeshclient.MemoryOpsDigestRequest{TenantID: "  "}); err == nil {
		t.Fatal("expected tenant_id required for whitespace")
	}

	// Empty window/horizon → client defaults day/ops.
	resp, err := nc.ExportOpsDigest(ctx, iomeshclient.MemoryOpsDigestRequest{TenantID: "dept.defaults"})
	if err != nil {
		t.Fatalf("ExportOpsDigest defaults: %v", err)
	}
	if resp.Window != "day" || resp.Horizon != "ops" {
		t.Fatalf("decoded window/horizon = %s/%s", resp.Window, resp.Horizon)
	}
	mu.Lock()
	if gotBody["window"] != "day" || gotBody["horizon"] != "ops" {
		t.Fatalf("request defaults body=%v", gotBody)
	}
	if _, ok := gotBody["limit"]; ok {
		t.Fatalf("limit should be omitted when zero, got %v", gotBody["limit"])
	}
	if _, ok := gotBody["as_of"]; ok {
		t.Fatalf("as_of should be omitted when empty, got %v", gotBody["as_of"])
	}
	mu.Unlock()
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
	if resp.Path != "/v5/memory/ingest" {
		t.Fatalf("path = %q", resp.Path)
	}
	if !resp.PalaceWrite() {
		t.Fatal("PalaceWrite() = false, want true for sidecar ok write")
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

func TestIngestMemoryTurnBrokerAcceptedStubNotPalaceWrite(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v5/memory/ingest", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "accepted",
			"note":   "memory ingest gated; sidecar proxy not configured on broker",
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: ts.URL})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	resp, err := nc.IngestMemoryTurn(context.Background(), "dept.research", iomeshclient.MemoryEnvelope{
		Content: "notes",
	})
	if err != nil {
		t.Fatalf("IngestMemoryTurn: %v", err)
	}
	if resp.Status != "accepted" {
		t.Fatalf("status = %q", resp.Status)
	}
	if resp.Note == "" {
		t.Fatal("expected note from broker stub")
	}
	if resp.PalaceWrite() {
		t.Fatal("PalaceWrite() = true for accepted stub — invents live APPLY")
	}
}

func TestMemoryIngestResponsePalaceWrite(t *testing.T) {
	if (iomeshclient.MemoryIngestResponse{}).PalaceWrite() {
		t.Fatal("empty PalaceWrite")
	}
	if (iomeshclient.MemoryIngestResponse{Status: "accepted", Note: "stub"}).PalaceWrite() {
		t.Fatal("accepted stub")
	}
	if !(iomeshclient.MemoryIngestResponse{Status: "ok", MemoryID: "m1"}).PalaceWrite() {
		t.Fatal("ok+id")
	}
	if !(iomeshclient.MemoryIngestResponse{Ingested: 2}).PalaceWrite() {
		t.Fatal("ingested count")
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

func TestDualWriteMemoryTurn_AsyncOnly(t *testing.T) {
	var published int
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/streams/MEMORY_INGEST/publish", func(w http.ResponseWriter, r *http.Request) {
		published++
		_ = json.NewEncoder(w).Encode(map[string]any{"stream": "MEMORY_INGEST", "seq": 1})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: ts.URL})
	if err != nil {
		t.Fatal(err)
	}
	res, err := nc.DualWriteMemoryTurn(context.Background(), "dept.x", iomeshclient.MemoryEnvelope{
		Role: "user", Content: "note", SessionID: "s1", SessionSeq: 1,
	}, iomeshclient.DualWriteMemoryOptions{Sync: false})
	if err != nil {
		t.Fatal(err)
	}
	if res.Async == nil || res.Async.Seq != 1 || res.Sync != nil || res.SyncErr != nil {
		t.Fatalf("%+v", res)
	}
	if published != 1 {
		t.Fatalf("published=%d", published)
	}
}

func TestDualWriteMemoryTurn_AsyncAndSync(t *testing.T) {
	var mu sync.Mutex
	var syncHits, asyncHits int
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/streams/MEMORY_INGEST/publish", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		asyncHits++
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"stream": "MEMORY_INGEST", "seq": 3})
	})
	mux.HandleFunc("POST /v1/memory/ingest", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		syncHits++
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "memory_id": "m1"})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: ts.URL})
	if err != nil {
		t.Fatal(err)
	}
	res, err := nc.DualWriteMemoryTurn(context.Background(), "dept.x", iomeshclient.MemoryEnvelope{
		Role: "assistant", Content: "dual", SessionID: "s2",
	}, iomeshclient.DualWriteMemoryOptions{Sync: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Async == nil || res.Async.Seq != 3 {
		t.Fatalf("async=%+v", res.Async)
	}
	if res.SyncErr != nil || res.Sync == nil || res.Sync.MemoryID != "m1" {
		t.Fatalf("sync=%+v err=%v", res.Sync, res.SyncErr)
	}
	mu.Lock()
	defer mu.Unlock()
	if asyncHits != 1 || syncHits != 1 {
		t.Fatalf("async=%d sync=%d", asyncHits, syncHits)
	}
}

func TestDualWriteMemoryTurn_SyncFailOpen(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/streams/MEMORY_INGEST/publish", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"stream": "MEMORY_INGEST", "seq": 2})
	})
	// no sync ingest routes → fail-open SyncErr
	ts := httptest.NewServer(mux)
	defer ts.Close()
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: ts.URL})
	if err != nil {
		t.Fatal(err)
	}
	res, err := nc.DualWriteMemoryTurn(context.Background(), "dept.x", iomeshclient.MemoryEnvelope{
		Content: "x",
	}, iomeshclient.DualWriteMemoryOptions{Sync: true})
	if err != nil {
		t.Fatalf("async must succeed: %v", err)
	}
	if res.Async == nil || res.SyncErr == nil {
		t.Fatalf("expected SyncErr fail-open, got %+v", res)
	}
}
