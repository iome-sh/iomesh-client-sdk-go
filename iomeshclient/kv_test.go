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

func TestCreateBucket_201Body(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	maxBytes := int64(1024)
	ttl := int64(3600)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if r.Method != http.MethodPost || r.URL.Path != "/v1/kv/agent-state" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":        "agent-state",
			"max_bytes":   1024,
			"history":     5,
			"ttl_seconds": 3600,
		})
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	info, err := nc.CreateBucket(context.Background(), "agent-state", iomeshclient.CreateBucketConfig{
		MaxBytes:   &maxBytes,
		History:    5,
		TTLSeconds: &ttl,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/kv/agent-state" {
		t.Fatalf("method=%q path=%q", gotMethod, gotPath)
	}
	if gotBody["history"] != float64(5) {
		t.Fatalf("request body=%v", gotBody)
	}
	if info == nil || info.Name != "agent-state" {
		t.Fatalf("info=%+v", info)
	}
	if info.History != 5 {
		t.Fatalf("history=%d", info.History)
	}
	if info.MaxBytes == nil || *info.MaxBytes != 1024 {
		t.Fatalf("max_bytes=%v", info.MaxBytes)
	}
	if info.TTLSeconds == nil || *info.TTLSeconds != 3600 {
		t.Fatalf("ttl=%v", info.TTLSeconds)
	}
}

func TestCreateBucket_409NameOnly(t *testing.T) {
	var posts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/kv/agent-state" {
			posts++
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"bucket already exists"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	info, err := nc.CreateBucket(context.Background(), "agent-state")
	if err != nil {
		t.Fatal(err)
	}
	if posts != 1 {
		t.Fatalf("posts=%d", posts)
	}
	if info == nil || info.Name != "agent-state" {
		t.Fatalf("info=%+v", info)
	}
	// 409 path does not populate optional fields
	if info.History != 0 || info.MaxBytes != nil || info.TTLSeconds != nil {
		t.Fatalf("expected name-only info, got %+v", info)
	}
}

func TestCreateBucket_Validation(t *testing.T) {
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: "http://127.0.0.1:9"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = nc.CreateBucket(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "bucket name required") {
		t.Fatalf("empty name err=%v", err)
	}

	_, err = nc.CreateBucket(context.Background(), "   ")
	if err == nil || !strings.Contains(err.Error(), "bucket name required") {
		t.Fatalf("whitespace name err=%v", err)
	}

	var c *iomeshclient.Client
	_, err = c.CreateBucket(context.Background(), "x")
	if err == nil || !strings.Contains(err.Error(), "nil client") {
		t.Fatalf("nil client err=%v", err)
	}
}

func TestCreateBucket_201OmitsName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"history": 1,
		})
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	info, err := nc.CreateBucket(context.Background(), "fallback-name")
	if err != nil {
		t.Fatal(err)
	}
	if info == nil || info.Name != "fallback-name" {
		t.Fatalf("expected defensive name, got %+v", info)
	}
}

func TestFormatKVEntry_Fields(t *testing.T) {
	out := iomeshclient.FormatKVEntry(iomeshclient.KVEntry{
		Bucket:    "agent-state",
		Key:       "worker-1.checkpoint",
		Value:     []byte("seq=42"),
		Revision:  3,
		CreatedAt: time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
	})
	for _, want := range []string{
		"iomesh kv entry",
		"bucket:     agent-state",
		"key:        worker-1.checkpoint",
		"revision:   3",
		"2026-07-01T12:00:00Z",
		"seq=42",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestFormatKVEntry_BinaryValue(t *testing.T) {
	out := iomeshclient.FormatKVEntry(iomeshclient.KVEntry{
		Bucket: "b",
		Key:    "k",
		Value:  []byte{0x00, 0x01, 0xff},
	})
	if !strings.Contains(out, "3 bytes") {
		t.Fatalf("binary value note missing: %q", out)
	}
}

func TestFormatKVKeys_Empty(t *testing.T) {
	out := iomeshclient.FormatKVKeys("agent-state", nil)
	if !strings.Contains(out, "bucket=agent-state") || !strings.Contains(out, "count=0") {
		t.Fatalf("empty: %q", out)
	}
	if !strings.Contains(out, "(no keys)") {
		t.Fatalf("no-keys marker: %q", out)
	}
	out = iomeshclient.FormatKVKeys("agent-state", []string{})
	if !strings.Contains(out, "count=0") || !strings.Contains(out, "(no keys)") {
		t.Fatalf("empty slice: %q", out)
	}
}

func TestFormatKVKeys_List(t *testing.T) {
	out := iomeshclient.FormatKVKeys("agent-state", []string{
		"worker-1.checkpoint",
		"worker-2.checkpoint",
	})
	if !strings.Contains(out, "count=2") {
		t.Fatalf("count: %q", out)
	}
	if !strings.Contains(out, "worker-1.checkpoint") || !strings.Contains(out, "worker-2.checkpoint") {
		t.Fatalf("keys: %q", out)
	}
	if !strings.HasPrefix(out, "iomesh kv keys bucket=agent-state") {
		t.Fatalf("header: %q", out)
	}
}
