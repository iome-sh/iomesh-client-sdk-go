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

func TestCreateStream_201Body(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if r.Method != http.MethodPost || r.URL.Path != "/v1/streams" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":       "EVENTS",
			"subjects":   []string{"dept.events.>"},
			"retention":  "limits",
			"partitions": 1,
			"messages":   0,
			"first_seq":  0,
			"last_seq":   0,
			"created_at": time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		})
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	info, err := nc.CreateStream(context.Background(), iomeshclient.StreamConfig{
		Name:     "EVENTS",
		Subjects: []string{"dept.events.>"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/streams" {
		t.Fatalf("method=%q path=%q", gotMethod, gotPath)
	}
	if info == nil || info.Name != "EVENTS" {
		t.Fatalf("info=%+v", info)
	}
	if len(info.Subjects) != 1 || info.Subjects[0] != "dept.events.>" {
		t.Fatalf("subjects=%v", info.Subjects)
	}
	if info.Retention != "limits" || info.Partitions != 1 {
		t.Fatalf("retention/partitions=%+v", info)
	}
}

func TestCreateStream_409ThenGetInfo(t *testing.T) {
	var posts, gets int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/streams":
			posts++
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"stream already exists"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/streams/EVENTS":
			gets++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name":      "EVENTS",
				"subjects":  []string{"dept.events.>"},
				"messages":  5,
				"first_seq": 1,
				"last_seq":  5,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	info, err := nc.CreateStream(context.Background(), iomeshclient.StreamConfig{
		Name:     "EVENTS",
		Subjects: []string{"dept.events.>"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if posts != 1 || gets != 1 {
		t.Fatalf("posts=%d gets=%d", posts, gets)
	}
	if info == nil || info.Name != "EVENTS" || info.LastSeq != 5 {
		t.Fatalf("info=%+v", info)
	}
}

func TestCreateStream_409GetFailsNilInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/streams":
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"stream already exists"}`))
		case r.Method == http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"stream not found"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	info, err := nc.CreateStream(context.Background(), iomeshclient.StreamConfig{
		Name:     "EVENTS",
		Subjects: []string{"dept.events.>"},
	})
	if err != nil {
		t.Fatalf("expected nil err on 409+GET fail, got %v", err)
	}
	if info != nil {
		t.Fatalf("expected nil info, got %+v", info)
	}
}

func TestCreateStream_Validation(t *testing.T) {
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: "http://127.0.0.1:9"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = nc.CreateStream(context.Background(), iomeshclient.StreamConfig{
		Name:     "",
		Subjects: []string{"x.>"},
	})
	if err == nil || !strings.Contains(err.Error(), "stream name required") {
		t.Fatalf("empty name err=%v", err)
	}

	_, err = nc.CreateStream(context.Background(), iomeshclient.StreamConfig{
		Name:     "EVENTS",
		Subjects: nil,
	})
	if err == nil || !strings.Contains(err.Error(), "subjects required") {
		t.Fatalf("empty subjects err=%v", err)
	}

	var c *iomeshclient.Client
	_, err = c.CreateStream(context.Background(), iomeshclient.StreamConfig{
		Name: "X", Subjects: []string{"x.>"},
	})
	if err == nil || !strings.Contains(err.Error(), "nil client") {
		t.Fatalf("nil client err=%v", err)
	}
}

func TestEnsureStream_Delegates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":     "KV",
			"subjects": []string{"kv.>"},
			"messages": 0,
		})
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	info, err := nc.EnsureStream(context.Background(), iomeshclient.StreamConfig{
		Name:     "KV",
		Subjects: []string{"kv.>"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if info == nil || info.Name != "KV" {
		t.Fatalf("info=%+v", info)
	}
}

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

func TestDeleteStream_OK204AndUserAgent(t *testing.T) {
	var gotUA, gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotUA = r.Header.Get("User-Agent")
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/streams/foo" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if err := nc.DeleteStream(context.Background(), "foo"); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotPath != "/v1/streams/foo" {
		t.Fatalf("path=%q", gotPath)
	}
	if !strings.HasPrefix(gotUA, "iomesh-client-sdk-go/") {
		t.Fatalf("User-Agent=%q", gotUA)
	}
}

func TestDeleteStream_EmptyName(t *testing.T) {
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: "http://127.0.0.1:9"})
	if err != nil {
		t.Fatal(err)
	}
	err = nc.DeleteStream(context.Background(), "  ")
	if err == nil || !strings.Contains(err.Error(), "stream name required") {
		t.Fatalf("err=%v", err)
	}
}

func TestDeleteStream_404APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"stream not found"}`))
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	err = nc.DeleteStream(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *iomeshclient.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("err=%v (want APIError 404)", err)
	}
}

func TestDeleteStream_NilClient(t *testing.T) {
	var c *iomeshclient.Client
	err := c.DeleteStream(context.Background(), "foo")
	if err == nil || !strings.Contains(err.Error(), "nil client") {
		t.Fatalf("err=%v", err)
	}
}

func TestFormatStreams_Empty(t *testing.T) {
	out := iomeshclient.FormatStreams(nil)
	if !strings.Contains(out, "count=0") || !strings.Contains(out, "(no streams)") {
		t.Fatalf("empty: %q", out)
	}
	out = iomeshclient.FormatStreams([]iomeshclient.StreamInfo{})
	if !strings.Contains(out, "count=0") || !strings.Contains(out, "(no streams)") {
		t.Fatalf("empty slice: %q", out)
	}
}

func TestFormatStreams_One(t *testing.T) {
	out := iomeshclient.FormatStreams([]iomeshclient.StreamInfo{
		{
			Name:      "EVENTS",
			Subjects:  []string{"dept.events.>"},
			Messages:  3,
			FirstSeq:  1,
			LastSeq:   3,
			Retention: "limits",
		},
	})
	if !strings.Contains(out, "count=1") {
		t.Fatalf("count: %q", out)
	}
	if !strings.Contains(out, "EVENTS") || !strings.Contains(out, "dept.events.>") {
		t.Fatalf("name/subjects: %q", out)
	}
	if !strings.Contains(out, "NAME") || !strings.Contains(out, "MSGS") {
		t.Fatalf("header: %q", out)
	}
	if !strings.Contains(out, "limits") {
		t.Fatalf("retention: %q", out)
	}
}

func TestFormatStreamDetail_Fields(t *testing.T) {
	max := int64(1000)
	age := int64(3600)
	detail := iomeshclient.FormatStreamDetail(iomeshclient.StreamInfo{
		Name:        "EVENTS",
		Description: "ops events",
		Retention:   "limits",
		Partitions:  1,
		MaxMsgs:     &max,
		MaxAgeSec:   &age,
		Messages:    10,
		FirstSeq:    1,
		LastSeq:     10,
		CreatedAt:   time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		Subjects:    []string{"dept.events.>", "dept.ops.>"},
	})
	for _, want := range []string{
		"iomesh stream",
		"name:        EVENTS",
		"ops events",
		"retention:   limits",
		"partitions:  1",
		"max_msgs:    1000",
		"max_age_sec: 3600",
		"messages:    10",
		"first_seq:   1",
		"last_seq:    10",
		"2026-07-01T12:00:00Z",
		"dept.events.>",
		"dept.ops.>",
	} {
		if !strings.Contains(detail, want) {
			t.Fatalf("missing %q in:\n%s", want, detail)
		}
	}
}
