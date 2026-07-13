package aionclient_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iome-sh/iomesh-client-sdk-go/aionclient"
)

func TestPublishPartitionRequestJSON(t *testing.T) {
	var captured struct {
		Subject      string `json:"subject"`
		Payload      string `json:"payload"`
		Partition    int    `json:"partition"`
		PartitionKey string `json:"partition_key"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/streams/MOCK/publish" {
			t.Fatalf("path = %q, want publish path", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("Unmarshal() error: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"stream":"MOCK","seq":1,"subject":"events.created","partition":3,"timestamp":"2026-01-01T00:00:00Z"}`))
	}))
	t.Cleanup(srv.Close)

	nc, err := aionclient.Connect(aionclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}

	ack, err := nc.Publish(context.Background(), "MOCK", "events.created", []byte("payload"),
		aionclient.WithPartitionKey("tenant-b"),
		aionclient.WithPartition(1),
	)
	if err != nil {
		t.Fatalf("Publish() error: %v", err)
	}

	if captured.Subject != "events.created" {
		t.Fatalf("subject = %q, want events.created", captured.Subject)
	}
	if captured.PartitionKey != "tenant-b" {
		t.Fatalf("partition_key = %q, want tenant-b", captured.PartitionKey)
	}
	if captured.Partition != 1 {
		t.Fatalf("partition = %d, want 1", captured.Partition)
	}
	if ack.Partition != 3 {
		t.Fatalf("PubAck.Partition = %d, want 3", ack.Partition)
	}
}

func TestPublishPartitionKeyOnlyOmitsExplicitPartition(t *testing.T) {
	var captured map[string]json.RawMessage

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("Unmarshal() error: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"stream":"MOCK","seq":1,"subject":"events.created","partition":0,"timestamp":"2026-01-01T00:00:00Z"}`))
	}))
	t.Cleanup(srv.Close)

	nc, err := aionclient.Connect(aionclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}

	if _, err := nc.Publish(context.Background(), "MOCK", "events.created", []byte("x"), aionclient.WithPartitionKey("k")); err != nil {
		t.Fatalf("Publish() error: %v", err)
	}
	if _, ok := captured["partition"]; ok {
		t.Fatal("partition field sent without WithPartition")
	}
	if string(captured["partition_key"]) != `"k"` {
		t.Fatalf("partition_key = %s, want %q", captured["partition_key"], "k")
	}
}