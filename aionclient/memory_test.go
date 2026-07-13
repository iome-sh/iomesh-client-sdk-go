package aionclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/iome-sh/iomesh-client-sdk-go/aionclient"
)

func TestRegisterMemoryProductAndPublishIngest(t *testing.T) {
	var mu sync.Mutex
	var created []aionclient.MemoryProductConfig
	var published []map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v5/registry/memory-products", func(w http.ResponseWriter, r *http.Request) {
		var cfg aionclient.MemoryProductConfig
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

	nc, err := aionclient.Connect(aionclient.Options{URL: ts.URL}, aionclient.WithTenant("dept.research"))
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}

	ctx := context.Background()
	if err := nc.RegisterMemoryProduct(ctx, aionclient.MemoryProductConfig{
		ProductID:  "research.memory",
		TenantID:   "dept.research",
		PalaceRoot: "aion-memory/dept.research/palace",
	}); err != nil {
		t.Fatalf("RegisterMemoryProduct: %v", err)
	}

	if _, err := nc.PublishMemoryIngest(ctx, "dept.research", aionclient.MemoryEnvelope{
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
