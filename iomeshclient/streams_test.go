package iomeshclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
)

func TestListStreams_OKArrayAndUserAgent(t *testing.T) {
	var gotUA, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotUA = r.Header.Get("User-Agent")
		if r.URL.Path != "/v1/streams" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"name":       "EVENTS",
				"subjects":   []string{"dept.events.>"},
				"messages":   3,
				"first_seq":  1,
				"last_seq":   3,
				"created_at": time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
			},
		})
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	streams, err := nc.ListStreams(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("method=%q", gotMethod)
	}
	if !strings.HasPrefix(gotUA, "iomesh-client-sdk-go/") {
		t.Fatalf("User-Agent=%q", gotUA)
	}
	if len(streams) != 1 || streams[0].Name != "EVENTS" {
		t.Fatalf("streams=%+v", streams)
	}
	if streams[0].Messages != 3 || streams[0].LastSeq != 3 {
		t.Fatalf("stats=%+v", streams[0])
	}
}

func TestListStreams_Envelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"streams": []map[string]any{
				{"name": "KV", "subjects": []string{"kv.>"}, "messages": 0, "first_seq": 0, "last_seq": 0},
			},
		})
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	streams, err := nc.ListStreams(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 || streams[0].Name != "KV" {
		t.Fatalf("streams=%+v", streams)
	}
}

func TestGetStream_OK(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodGet || r.URL.Path != "/v1/streams/EVENTS" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":       "EVENTS",
			"subjects":   []string{"dept.events.>"},
			"retention":  "limits",
			"partitions": 1,
			"messages":   10,
			"first_seq":  1,
			"last_seq":   10,
			"created_at": time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		})
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	info, err := nc.GetStream(context.Background(), "EVENTS")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/streams/EVENTS" {
		t.Fatalf("path=%q", gotPath)
	}
	if info == nil || info.Name != "EVENTS" || info.LastSeq != 10 {
		t.Fatalf("info=%+v", info)
	}
}

func TestGetStream_404APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"stream not found"}`))
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = nc.GetStream(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *iomeshclient.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("err=%v (want APIError 404)", err)
	}
}

func TestGetStream_EmptyName(t *testing.T) {
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: "http://127.0.0.1:9"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = nc.GetStream(context.Background(), "  ")
	if err == nil || !strings.Contains(err.Error(), "stream name required") {
		t.Fatalf("err=%v", err)
	}
}

func TestListStreams_NilClient(t *testing.T) {
	var c *iomeshclient.Client
	_, err := c.ListStreams(context.Background())
	if err == nil || !strings.Contains(err.Error(), "nil client") {
		t.Fatalf("err=%v", err)
	}
}

func TestGetStream_NilClient(t *testing.T) {
	var c *iomeshclient.Client
	_, err := c.GetStream(context.Background(), "EVENTS")
	if err == nil || !strings.Contains(err.Error(), "nil client") {
		t.Fatalf("err=%v", err)
	}
}
