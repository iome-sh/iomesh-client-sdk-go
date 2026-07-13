//go:build ignore

package aionclient_test

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	httpapi "github.com/iome-sh/aion/internal/api/http"
	"github.com/iome-sh/aion/internal/broker"
	"github.com/iome-sh/aion/internal/consumer"
	"github.com/iome-sh/aion/internal/domain"
	"github.com/iome-sh/aion/internal/ephemeral"
	"github.com/iome-sh/aion/internal/protocol/kafka"
	"github.com/iome-sh/aion/internal/registry"
	"github.com/iome-sh/aion/internal/storage/mem"
	"github.com/iome-sh/aion/pkg/aionclient"
)

func newMeshSDKHTTPBrokerURL(t *testing.T) string {
	t.Helper()

	store := mem.New()
	t.Cleanup(func() { _ = store.Close() })

	b := broker.New(store)
	cm := consumer.NewManager(store, nil)
	hub := ephemeral.New()

	mux := http.NewServeMux()
	httpapi.RegisterRoutes(mux, b, cm, hub, store, nil, nil, "", false, nil)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts.URL
}

func TestMeshSDKPub(t *testing.T) {
	t.Parallel()

	sdk, err := aionclient.NewMeshSDK(aionclient.MeshSDKConfig{BrokerURL: newMeshSDKHTTPBrokerURL(t)})
	if err != nil {
		t.Fatalf("NewMeshSDK: %v", err)
	}

	if err := sdk.Pub(context.Background(), "agent.events.worker.started", []byte("started"), map[string]string{
		"trace": "mesh-sdk",
	}); err != nil {
		t.Fatalf("Pub: %v", err)
	}
}

func TestMeshSDKPublish(t *testing.T) {
	t.Parallel()

	sdk, err := aionclient.NewMeshSDK(aionclient.MeshSDKConfig{BrokerURL: newMeshSDKHTTPBrokerURL(t)})
	if err != nil {
		t.Fatalf("NewMeshSDK: %v", err)
	}

	ctx := context.Background()
	if err := sdk.EnsureStream(ctx, aionclient.StreamConfig{
		Name:     "MESH_SDK",
		Subjects: []string{"mesh.sdk.>"},
	}); err != nil {
		t.Fatalf("EnsureStream: %v", err)
	}
	ack, err := sdk.Publish(ctx, "MESH_SDK", "mesh.sdk.events.created", []byte("evt"))
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if ack == nil || ack.Stream != "MESH_SDK" {
		t.Fatalf("ack = %+v", ack)
	}
}

func TestMeshSDKKafkaProduce(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := mem.New()
	t.Cleanup(func() { _ = store.Close() })

	reg := registry.NewFakeStore()
	b := broker.New(store)
	tenant := "dept.engineering"
	if err := b.Streams().Create(ctx, domain.StreamConfig{
		Name:     "EVENTS",
		Subjects: []string{"dept.engineering.events.>"},
	}); err != nil {
		t.Fatalf("Create stream: %v", err)
	}
	if err := reg.RegisterKafkaMapping(ctx, domain.KafkaTopicMapping{
		Topic: "events.raw", Stream: "EVENTS",
		Subject: "dept.engineering.events.github", Tenant: tenant,
	}); err != nil {
		t.Fatalf("RegisterKafkaMapping: %v", err)
	}

	mappings := kafka.NewRegistryMappingStore(reg, tenant)
	backend := kafka.NewBrokerBackend(b, mappings)
	srv := kafka.NewServer(backend)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })

	sdk, err := aionclient.NewMeshSDK(aionclient.MeshSDKConfig{
		BrokerURL: "http://127.0.0.1:1",
		KafkaAddr: ln.Addr().String(),
		Tenant:    tenant,
	})
	if err != nil {
		t.Fatalf("NewMeshSDK: %v", err)
	}
	t.Cleanup(func() { _ = sdk.Close() })

	payload, _ := json.Marshal(map[string]any{"event_id": "evt-mesh-sdk"})
	offset, err := sdk.KafkaProduce(ctx, "events.raw", 0, []byte("key"), payload)
	if err != nil {
		t.Fatalf("KafkaProduce: %v", err)
	}
	if offset <= 0 {
		t.Fatalf("offset = %d, want > 0", offset)
	}
}

func TestMeshSDKKafkaProduceRequiresAddr(t *testing.T) {
	t.Parallel()

	sdk, err := aionclient.NewMeshSDK(aionclient.MeshSDKConfig{BrokerURL: "http://127.0.0.1:8422"})
	if err != nil {
		t.Fatalf("NewMeshSDK: %v", err)
	}

	_, err = sdk.KafkaProduce(context.Background(), "events.raw", 0, nil, []byte("x"))
	if err == nil {
		t.Fatal("expected KafkaAddr not configured error")
	}
}
