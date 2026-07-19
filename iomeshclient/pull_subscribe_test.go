package iomeshclient_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
