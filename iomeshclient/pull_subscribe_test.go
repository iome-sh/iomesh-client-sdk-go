package iomeshclient_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
)

func TestCreateConsumer_201FullInfo(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if r.Method != http.MethodPost || r.URL.Path != "/v1/streams/EVENTS/consumers" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"stream":         "EVENTS",
			"name":           "worker-1",
			"ack_floor":      42,
			"pending_count":  3,
			"filter_subject": "dept.events.>",
		})
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	info, err := nc.CreateConsumer(context.Background(), iomeshclient.CreateConsumerConfig{
		Stream:        "EVENTS",
		Name:          "worker-1",
		FilterSubject: "dept.events.>",
		MaxDeliver:    5,
		AckWaitSec:    30,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/streams/EVENTS/consumers" {
		t.Fatalf("method=%q path=%q", gotMethod, gotPath)
	}
	if gotBody["name"] != "worker-1" {
		t.Fatalf("request body=%v", gotBody)
	}
	if gotBody["filter_subject"] != "dept.events.>" {
		t.Fatalf("filter in body=%v", gotBody)
	}
	if gotBody["max_deliver"] != float64(5) {
		t.Fatalf("max_deliver in body=%v", gotBody)
	}
	if gotBody["ack_wait_sec"] != float64(30) {
		t.Fatalf("ack_wait_sec in body=%v", gotBody)
	}
	if info == nil {
		t.Fatal("info is nil")
	}
	if info.Stream != "EVENTS" || info.Name != "worker-1" {
		t.Fatalf("info stream/name=%+v", info)
	}
	if info.AckFloor != 42 {
		t.Fatalf("ack_floor=%d", info.AckFloor)
	}
	if info.PendingCount != 3 {
		t.Fatalf("pending_count=%d", info.PendingCount)
	}
	if info.FilterSubject != "dept.events.>" {
		t.Fatalf("filter_subject=%q", info.FilterSubject)
	}
}

func TestCreateConsumer_409NameOnly(t *testing.T) {
	var posts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/streams/EVENTS/consumers" {
			posts++
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"consumer already exists"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	info, err := nc.CreateConsumer(context.Background(), iomeshclient.CreateConsumerConfig{
		Stream: "EVENTS",
		Name:   "worker-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if posts != 1 {
		t.Fatalf("posts=%d", posts)
	}
	if info == nil {
		t.Fatal("info is nil")
	}
	if info.Stream != "EVENTS" || info.Name != "worker-1" {
		t.Fatalf("expected Stream/Name only, got %+v", info)
	}
	if info.AckFloor != 0 || info.PendingCount != 0 || info.FilterSubject != "" {
		t.Fatalf("expected name-only on 409, got %+v", info)
	}
}

func TestCreateConsumer_Validation(t *testing.T) {
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: "http://127.0.0.1:9"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = nc.CreateConsumer(context.Background(), iomeshclient.CreateConsumerConfig{})
	if err == nil || !strings.Contains(err.Error(), "stream and name required") {
		t.Fatalf("empty cfg err=%v", err)
	}
	_, err = nc.CreateConsumer(context.Background(), iomeshclient.CreateConsumerConfig{Stream: "S"})
	if err == nil || !strings.Contains(err.Error(), "stream and name required") {
		t.Fatalf("missing name err=%v", err)
	}
	_, err = nc.CreateConsumer(context.Background(), iomeshclient.CreateConsumerConfig{Name: "c"})
	if err == nil || !strings.Contains(err.Error(), "stream and name required") {
		t.Fatalf("missing stream err=%v", err)
	}
	var c *iomeshclient.Client
	_, err = c.CreateConsumer(context.Background(), iomeshclient.CreateConsumerConfig{Stream: "S", Name: "c"})
	if err == nil || !strings.Contains(err.Error(), "nil client") {
		t.Fatalf("nil client err=%v", err)
	}
}

func TestEnsureConsumer_409NameOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/streams/EVENTS/consumers" {
			w.WriteHeader(http.StatusConflict)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	info, err := nc.EnsureConsumer(context.Background(), iomeshclient.CreateConsumerConfig{
		Stream: "EVENTS",
		Name:   "worker-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if info == nil || info.Stream != "EVENTS" || info.Name != "worker-1" {
		t.Fatalf("info=%+v", info)
	}
}

func TestCreateConsumer_PathEscape(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"stream": "a/b",
			"name":   "c",
		})
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	info, err := nc.CreateConsumer(context.Background(), iomeshclient.CreateConsumerConfig{
		Stream: "a/b",
		Name:   "c",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/streams/a%2Fb/consumers" {
		t.Fatalf("path=%q want escaped stream", gotPath)
	}
	if info.Stream != "a/b" || info.Name != "c" {
		t.Fatalf("info=%+v", info)
	}
}

func TestDeleteConsumer_OK204AndUserAgent(t *testing.T) {
	var gotUA, gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotUA = r.Header.Get("User-Agent")
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/streams/EVENTS/consumers/worker-1" {
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
	if err := nc.DeleteConsumer(context.Background(), "EVENTS", "worker-1"); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotPath != "/v1/streams/EVENTS/consumers/worker-1" {
		t.Fatalf("path=%q", gotPath)
	}
	if !strings.HasPrefix(gotUA, "iomesh-client-sdk-go/") {
		t.Fatalf("User-Agent=%q", gotUA)
	}
}

func TestDeleteConsumer_EmptyName(t *testing.T) {
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: "http://127.0.0.1:9"})
	if err != nil {
		t.Fatal(err)
	}
	err = nc.DeleteConsumer(context.Background(), "EVENTS", "  ")
	if err == nil || !strings.Contains(err.Error(), "stream and name required") {
		t.Fatalf("empty name err=%v", err)
	}
	err = nc.DeleteConsumer(context.Background(), "  ", "worker-1")
	if err == nil || !strings.Contains(err.Error(), "stream and name required") {
		t.Fatalf("empty stream err=%v", err)
	}
	err = nc.DeleteConsumer(context.Background(), "", "")
	if err == nil || !strings.Contains(err.Error(), "stream and name required") {
		t.Fatalf("both empty err=%v", err)
	}
}

func TestDeleteConsumer_404APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"consumer not found"}`))
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	err = nc.DeleteConsumer(context.Background(), "EVENTS", "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *iomeshclient.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("err=%v (want APIError 404)", err)
	}
}

func TestDeleteConsumer_NilClient(t *testing.T) {
	var c *iomeshclient.Client
	err := c.DeleteConsumer(context.Background(), "EVENTS", "worker-1")
	if err == nil || !strings.Contains(err.Error(), "nil client") {
		t.Fatalf("err=%v", err)
	}
}

func TestDeleteConsumer_PathEscape(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if err := nc.DeleteConsumer(context.Background(), "a/b", "c/d"); err != nil {
		t.Fatal(err)
	}
	// url.PathEscape("a/b") => "a%2Fb"; url.PathEscape("c/d") => "c%2Fd"
	if gotPath != "/v1/streams/a%2Fb/consumers/c%2Fd" {
		t.Fatalf("path=%q want escaped stream and name", gotPath)
	}
}

func TestSubscription_Delete_OK204(t *testing.T) {
	var gotMethod, gotPath string
	var createHits, deleteHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/streams/EVENTS/consumers":
			createHits++
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"stream": "EVENTS",
				"name":   "worker-1",
			})
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/streams/EVENTS/consumers/worker-1":
			deleteHits++
			gotMethod = r.Method
			gotPath = r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := nc.PullSubscribe(context.Background(), iomeshclient.PullSubscribeConfig{
		Stream:   "EVENTS",
		Consumer: "worker-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := sub.Delete(context.Background()); err != nil {
		t.Fatal(err)
	}
	if createHits != 1 || deleteHits != 1 {
		t.Fatalf("createHits=%d deleteHits=%d", createHits, deleteHits)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotPath != "/v1/streams/EVENTS/consumers/worker-1" {
		t.Fatalf("path=%q", gotPath)
	}
}

func TestSubscription_Delete_NilSubscription(t *testing.T) {
	var sub *iomeshclient.Subscription
	err := sub.Delete(context.Background())
	if err == nil || !strings.Contains(err.Error(), "nil subscription") {
		t.Fatalf("err=%v", err)
	}

	// Non-nil handle with nil client (e.g. zero value).
	sub = &iomeshclient.Subscription{}
	err = sub.Delete(context.Background())
	if err == nil || !strings.Contains(err.Error(), "nil subscription") {
		t.Fatalf("nil client err=%v", err)
	}
}

func TestSubscription_Delete_PathEscape(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/consumers"):
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"stream": "a/b", "name": "c/d"})
		case r.Method == http.MethodDelete:
			gotPath = r.URL.EscapedPath()
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := nc.PullSubscribe(context.Background(), iomeshclient.PullSubscribeConfig{
		Stream:   "a/b",
		Consumer: "c/d",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := sub.Delete(context.Background()); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/streams/a%2Fb/consumers/c%2Fd" {
		t.Fatalf("path=%q want escaped stream and name", gotPath)
	}
}

func TestPullSubscribe_201SetsConsumerInfo(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if r.Method != http.MethodPost || r.URL.Path != "/v1/streams/EVENTS/consumers" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"stream":         "EVENTS",
			"name":           "worker-1",
			"ack_floor":      42,
			"pending_count":  3,
			"filter_subject": "dept.events.>",
		})
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := nc.PullSubscribe(context.Background(), iomeshclient.PullSubscribeConfig{
		Stream:   "EVENTS",
		Consumer: "worker-1",
		Filter:   "dept.events.>",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/streams/EVENTS/consumers" {
		t.Fatalf("method=%q path=%q", gotMethod, gotPath)
	}
	if gotBody["name"] != "worker-1" {
		t.Fatalf("request body=%v", gotBody)
	}
	if gotBody["filter_subject"] != "dept.events.>" {
		t.Fatalf("filter in body=%v", gotBody)
	}

	info := sub.ConsumerInfo()
	if info.Stream != "EVENTS" || info.Name != "worker-1" {
		t.Fatalf("info stream/name=%+v", info)
	}
	if info.AckFloor != 42 {
		t.Fatalf("ack_floor=%d", info.AckFloor)
	}
	if info.PendingCount != 3 {
		t.Fatalf("pending_count=%d", info.PendingCount)
	}
	if info.FilterSubject != "dept.events.>" {
		t.Fatalf("filter_subject=%q", info.FilterSubject)
	}
}

func TestPullSubscribe_409NameOnlyInfo(t *testing.T) {
	var posts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/streams/EVENTS/consumers" {
			posts++
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"consumer already exists"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := nc.PullSubscribe(context.Background(), iomeshclient.PullSubscribeConfig{
		Stream:   "EVENTS",
		Consumer: "worker-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if posts != 1 {
		t.Fatalf("posts=%d", posts)
	}
	info := sub.ConsumerInfo()
	// 409 path: Stream/Name only (via CreateConsumer)
	if info.Stream != "EVENTS" || info.Name != "worker-1" {
		t.Fatalf("expected Stream/Name on 409, got %+v", info)
	}
	if info.AckFloor != 0 || info.PendingCount != 0 || info.FilterSubject != "" {
		t.Fatalf("expected name-only on 409, got %+v", info)
	}
}

func TestPullSubscribe_PathEscape(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"stream": "a/b",
			"name":   "c",
		})
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := nc.PullSubscribe(context.Background(), iomeshclient.PullSubscribeConfig{
		Stream:   "a/b",
		Consumer: "c",
	})
	if err != nil {
		t.Fatal(err)
	}
	// url.PathEscape("a/b") => "a%2Fb"
	if gotPath != "/v1/streams/a%2Fb/consumers" {
		t.Fatalf("path=%q want escaped stream", gotPath)
	}
	_ = sub
}

func TestSubscription_FetchAckNack_PathEscape(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.EscapedPath())
		switch {
		case strings.HasSuffix(r.URL.Path, "/consumers"):
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"stream": "a/b", "name": "c/d"})
		case strings.HasSuffix(r.URL.Path, "/fetch"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"messages": []map[string]any{
					{
						"stream":  "a/b",
						"seq":     1,
						"subject": "x",
						"payload": base64.StdEncoding.EncodeToString([]byte("hi")),
						"headers": map[string]string{},
					},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/ack"), strings.HasSuffix(r.URL.Path, "/nack"):
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := nc.PullSubscribe(context.Background(), iomeshclient.PullSubscribeConfig{
		Stream:   "a/b",
		Consumer: "c/d",
	})
	if err != nil {
		t.Fatal(err)
	}
	msgs, err := sub.Fetch(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("msgs=%d", len(msgs))
	}
	if err := sub.Ack(1); err != nil {
		t.Fatal(err)
	}
	if err := sub.Nack(1); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"/v1/streams/a%2Fb/consumers",
		"/v1/streams/a%2Fb/consumers/c%2Fd/fetch",
		"/v1/streams/a%2Fb/consumers/c%2Fd/ack",
		"/v1/streams/a%2Fb/consumers/c%2Fd/nack",
	}
	if len(paths) != len(want) {
		t.Fatalf("paths=%v want %v", paths, want)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("path[%d]=%q want %q", i, paths[i], want[i])
		}
	}
}

func TestSubscription_FetchContext_MaxWait(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/consumers"):
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"stream": "EVENTS", "name": "worker-1"})
		case strings.HasSuffix(r.URL.Path, "/fetch"):
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"messages": []map[string]any{
					{
						"stream":  "EVENTS",
						"seq":     9,
						"subject": "dept.events.x",
						"payload": base64.StdEncoding.EncodeToString([]byte("hi")),
						"headers": map[string]string{},
					},
				},
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
	sub, err := nc.PullSubscribe(context.Background(), iomeshclient.PullSubscribeConfig{
		Stream: "EVENTS", Consumer: "worker-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Custom MaxWait should land in max_wait_ms.
	msgs, err := sub.FetchContext(context.Background(), 3, iomeshclient.MaxWait(1500*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Seq() != 9 {
		t.Fatalf("msgs=%v", msgs)
	}
	if gotBody["batch"] != float64(3) {
		t.Fatalf("batch=%v", gotBody["batch"])
	}
	if gotBody["max_wait_ms"] != float64(1500) {
		t.Fatalf("max_wait_ms=%v want 1500", gotBody["max_wait_ms"])
	}
	// Rebind: Msg.Ack should hit the caller's subscription (not ephemeral).
	if out := iomeshclient.FormatMsg(msgs[0]); !strings.Contains(out, "seq=9") || !strings.Contains(out, "bytes=2") {
		t.Fatalf("FormatMsg=%q", out)
	}
}

func TestSubscription_FetchContext_DefaultMaxWait(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/consumers"):
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"stream": "S", "name": "c"})
		case strings.HasSuffix(r.URL.Path, "/fetch"):
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_ = json.NewEncoder(w).Encode(map[string]any{"messages": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := nc.PullSubscribe(context.Background(), iomeshclient.PullSubscribeConfig{
		Stream: "S", Consumer: "c",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sub.FetchContext(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	wantMs := float64(iomeshclient.DefaultFetchMaxWait.Milliseconds())
	if gotBody["max_wait_ms"] != wantMs {
		t.Fatalf("max_wait_ms=%v want %v (DefaultFetchMaxWait)", gotBody["max_wait_ms"], wantMs)
	}
}

func TestSubscription_FetchContext_Canceled(t *testing.T) {
	// Already-canceled ctx must fail before/without needing a live long-poll.
	// Handler would hang if a request were fully sent; cancel is checked on Do.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/consumers"):
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"stream": "S", "name": "c"})
		case strings.HasSuffix(r.URL.Path, "/fetch"):
			// Should not be reached when ctx is already canceled; if it is, fail fast.
			http.Error(w, "unexpected fetch with canceled ctx", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := nc.PullSubscribe(context.Background(), iomeshclient.PullSubscribeConfig{
		Stream: "S", Consumer: "c",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = sub.FetchContext(ctx, 1)
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
	if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "canceled") && !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("err=%v", err)
	}
}

func TestSubscription_AckNackContext(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.EscapedPath())
		switch {
		case strings.HasSuffix(r.URL.Path, "/consumers"):
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"stream": "EVENTS", "name": "worker-1"})
		case strings.HasSuffix(r.URL.Path, "/ack"), strings.HasSuffix(r.URL.Path, "/nack"):
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := nc.PullSubscribe(context.Background(), iomeshclient.PullSubscribeConfig{
		Stream: "EVENTS", Consumer: "worker-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := sub.AckContext(context.Background(), 1, 2); err != nil {
		t.Fatal(err)
	}
	if err := sub.NackContext(context.Background(), 3); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"/v1/streams/EVENTS/consumers",
		"/v1/streams/EVENTS/consumers/worker-1/ack",
		"/v1/streams/EVENTS/consumers/worker-1/nack",
	}
	if len(paths) != len(want) {
		t.Fatalf("paths=%v", paths)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("path[%d]=%q want %q", i, paths[i], want[i])
		}
	}
}

func TestFormatMsg(t *testing.T) {
	if out := iomeshclient.FormatMsg(nil); out != "iomesh msg (nil)\n" {
		t.Fatalf("nil=%q", out)
	}
	// Build a real msg via fetch so fields are set correctly.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]any{
				{
					"stream":  "S",
					"seq":     42,
					"subject": "dept.events.x",
					"payload": base64.StdEncoding.EncodeToString([]byte("hello")),
					"headers": map[string]string{},
				},
			},
		})
	}))
	defer srv.Close()
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	msgs, err := nc.ConsumerFetch(context.Background(), "S", "c", 1)
	if err != nil {
		t.Fatal(err)
	}
	out := iomeshclient.FormatMsg(msgs[0])
	for _, want := range []string{"seq=42", "subject=dept.events.x", "bytes=5"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestFormatMsgs_Empty(t *testing.T) {
	if out := iomeshclient.FormatMsgs(nil); out != "iomesh msgs count=0\n" {
		t.Fatalf("nil=%q", out)
	}
	if out := iomeshclient.FormatMsgs([]*iomeshclient.Msg{}); out != "iomesh msgs count=0\n" {
		t.Fatalf("empty=%q", out)
	}
}

func TestFormatMsgs_Multi(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]any{
				{
					"stream":  "S",
					"seq":     1,
					"subject": "dept.events.a",
					"payload": base64.StdEncoding.EncodeToString([]byte("hi")),
					"headers": map[string]string{},
				},
				{
					"stream":  "S",
					"seq":     2,
					"subject": "dept.events.b",
					"payload": base64.StdEncoding.EncodeToString([]byte("xyz")),
					"headers": map[string]string{},
				},
			},
		})
	}))
	defer srv.Close()
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	msgs, err := nc.ConsumerFetch(context.Background(), "S", "c", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("len=%d", len(msgs))
	}
	out := iomeshclient.FormatMsgs(msgs)
	if !strings.HasPrefix(out, "iomesh msgs count=2\n") {
		t.Fatalf("header=%q", out)
	}
	for _, want := range []string{
		"seq=1", "subject=dept.events.a", "bytes=2",
		"seq=2", "subject=dept.events.b", "bytes=3",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	// Each message line is FormatMsg output.
	if !strings.Contains(out, iomeshclient.FormatMsg(msgs[0])) {
		t.Fatalf("missing FormatMsg[0] in %q", out)
	}
	if !strings.Contains(out, iomeshclient.FormatMsg(msgs[1])) {
		t.Fatalf("missing FormatMsg[1] in %q", out)
	}
}

func TestConsumerAck_PathAndBody(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if err := nc.ConsumerAck(context.Background(), "EVENTS", "worker-1", 10, 11); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotPath != "/v1/streams/EVENTS/consumers/worker-1/ack" {
		t.Fatalf("path=%q", gotPath)
	}
	seqs, ok := gotBody["seqs"].([]any)
	if !ok || len(seqs) != 2 {
		t.Fatalf("body=%v", gotBody)
	}
	if seqs[0] != float64(10) || seqs[1] != float64(11) {
		t.Fatalf("seqs=%v", seqs)
	}
}

func TestConsumerAck_PathEscape(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if err := nc.ConsumerAck(context.Background(), "a/b", "c/d", 1); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/streams/a%2Fb/consumers/c%2Fd/ack" {
		t.Fatalf("path=%q", gotPath)
	}
}

func TestConsumerAck_Validation(t *testing.T) {
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: "http://127.0.0.1:9"})
	if err != nil {
		t.Fatal(err)
	}
	err = nc.ConsumerAck(context.Background(), "", "c", 1)
	if err == nil || !strings.Contains(err.Error(), "stream and consumer required") {
		t.Fatalf("empty stream err=%v", err)
	}
	err = nc.ConsumerAck(context.Background(), "S", "c")
	if err == nil || !strings.Contains(err.Error(), "seqs required") {
		t.Fatalf("empty seqs err=%v", err)
	}
	var c *iomeshclient.Client
	err = c.ConsumerAck(context.Background(), "S", "c", 1)
	if err == nil || !strings.Contains(err.Error(), "nil client") {
		t.Fatalf("nil client err=%v", err)
	}
}

func TestConsumerNack_PathAndBody(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if err := nc.ConsumerNack(context.Background(), "EVENTS", "worker-1", 7); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/streams/EVENTS/consumers/worker-1/nack" {
		t.Fatalf("method=%q path=%q", gotMethod, gotPath)
	}
	seqs, ok := gotBody["seqs"].([]any)
	if !ok || len(seqs) != 1 || seqs[0] != float64(7) {
		t.Fatalf("body=%v", gotBody)
	}
}

func TestConsumerFetch(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]any{
				{
					"stream":  "EVENTS",
					"seq":     42,
					"subject": "dept.events.x",
					"payload": base64.StdEncoding.EncodeToString([]byte("hello")),
					"headers": map[string]string{"k": "v"},
				},
			},
		})
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	msgs, err := nc.ConsumerFetch(context.Background(), "EVENTS", "worker-1", 5)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/streams/EVENTS/consumers/worker-1/fetch" {
		t.Fatalf("method=%q path=%q", gotMethod, gotPath)
	}
	if gotBody["batch"] != float64(5) {
		t.Fatalf("body=%v", gotBody)
	}
	if len(msgs) != 1 {
		t.Fatalf("msgs=%d", len(msgs))
	}
	if msgs[0].Seq() != 42 {
		t.Fatalf("seq=%d", msgs[0].Seq())
	}
	if string(msgs[0].Data()) != "hello" {
		t.Fatalf("data=%q", msgs[0].Data())
	}
	if msgs[0].Subject() != "dept.events.x" {
		t.Fatalf("subject=%q", msgs[0].Subject())
	}
	if msgs[0].Headers()["k"] != "v" {
		t.Fatalf("headers=%v", msgs[0].Headers())
	}
}

func TestConsumerFetch_PathEscapeAndMsgAck(t *testing.T) {
	var paths []string
	var ackBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.EscapedPath())
		switch {
		case strings.HasSuffix(r.URL.Path, "/fetch"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"messages": []map[string]any{
					{
						"stream":  "a/b",
						"seq":     9,
						"subject": "x",
						"payload": base64.StdEncoding.EncodeToString([]byte("hi")),
						"headers": map[string]string{},
					},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/ack"):
			_ = json.NewDecoder(r.Body).Decode(&ackBody)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	msgs, err := nc.ConsumerFetch(context.Background(), "a/b", "c/d", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("msgs=%d", len(msgs))
	}
	// Ephemeral sub wiring: Msg.Ack must hit ConsumerAck path.
	if err := msgs[0].Ack(); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"/v1/streams/a%2Fb/consumers/c%2Fd/fetch",
		"/v1/streams/a%2Fb/consumers/c%2Fd/ack",
	}
	if len(paths) != len(want) {
		t.Fatalf("paths=%v want %v", paths, want)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("path[%d]=%q want %q", i, paths[i], want[i])
		}
	}
	seqs, ok := ackBody["seqs"].([]any)
	if !ok || len(seqs) != 1 || seqs[0] != float64(9) {
		t.Fatalf("ack body=%v", ackBody)
	}
}

func TestConsumerFetch_Validation(t *testing.T) {
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: "http://127.0.0.1:9"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = nc.ConsumerFetch(context.Background(), "", "c", 1)
	if err == nil || !strings.Contains(err.Error(), "stream and consumer required") {
		t.Fatalf("empty stream err=%v", err)
	}
	_, err = nc.ConsumerFetch(context.Background(), "S", "c", 0)
	if err == nil || !strings.Contains(err.Error(), "batch must be > 0") {
		t.Fatalf("batch err=%v", err)
	}
	var c *iomeshclient.Client
	_, err = c.ConsumerFetch(context.Background(), "S", "c", 1)
	if err == nil || !strings.Contains(err.Error(), "nil client") {
		t.Fatalf("nil client err=%v", err)
	}
}

func TestSubscription_AckStillWorks(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/streams/EVENTS/consumers" {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"stream": "EVENTS", "name": "worker-1"})
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/v1/streams/EVENTS/consumers/worker-1/ack" {
			gotPath = r.URL.Path
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := nc.PullSubscribe(context.Background(), iomeshclient.PullSubscribeConfig{
		Stream:   "EVENTS",
		Consumer: "worker-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := sub.Ack(99); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/streams/EVENTS/consumers/worker-1/ack" {
		t.Fatalf("path=%q", gotPath)
	}
	seqs, ok := gotBody["seqs"].([]any)
	if !ok || len(seqs) != 1 || seqs[0] != float64(99) {
		t.Fatalf("body=%v", gotBody)
	}
}

func TestPullSubscribe_Validation(t *testing.T) {
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: "http://127.0.0.1:9"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = nc.PullSubscribe(context.Background(), iomeshclient.PullSubscribeConfig{})
	if err == nil || !strings.Contains(err.Error(), "stream and consumer required") {
		t.Fatalf("empty cfg err=%v", err)
	}
	_, err = nc.PullSubscribe(context.Background(), iomeshclient.PullSubscribeConfig{Stream: "S"})
	if err == nil || !strings.Contains(err.Error(), "stream and consumer required") {
		t.Fatalf("missing consumer err=%v", err)
	}
}

func TestFormatConsumerInfo(t *testing.T) {
	out := iomeshclient.FormatConsumerInfo(iomeshclient.ConsumerInfo{
		Stream:        "EVENTS",
		Name:          "worker-1",
		AckFloor:      42,
		PendingCount:  3,
		FilterSubject: "dept.events.>",
	})
	for _, want := range []string{
		"iomesh consumer",
		"stream:          EVENTS",
		"name:            worker-1",
		"ack_floor:       42",
		"pending_count:   3",
		"filter_subject:  dept.events.>",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}

	// empty filter omitted
	out = iomeshclient.FormatConsumerInfo(iomeshclient.ConsumerInfo{
		Stream: "S",
		Name:   "c",
	})
	if strings.Contains(out, "filter_subject") {
		t.Fatalf("expected no filter_subject line:\n%s", out)
	}
	if !strings.Contains(out, "stream:          S") || !strings.Contains(out, "name:            c") {
		t.Fatalf("zero-ish info:\n%s", out)
	}
}
