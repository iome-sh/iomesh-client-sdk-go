//go:build ignore

package aionclient_test

import (
	"context"
	"testing"
	"time"

	"github.com/iome-sh/aion/internal/domain"
	"github.com/iome-sh/iomesh-client-sdk-go/aionclient"
)

func TestPublishWithPartitionKeyIntegration(t *testing.T) {
	nc := newTestClient(t)
	ctx := context.Background()

	const partitions = 4
	if err := nc.CreateStream(ctx, aionclient.StreamConfig{
		Name:       "PARTITIONED",
		Subjects:   []string{"events.>"},
		Partitions: partitions,
	}); err != nil {
		t.Fatalf("CreateStream() error: %v", err)
	}

	key := "tenant-a"
	wantPartition, err := domain.ResolvePartition(partitions, 0, key)
	if err != nil {
		t.Fatalf("ResolvePartition() error: %v", err)
	}

	ack, err := nc.Publish(ctx, "PARTITIONED", "events.created", []byte("payload"), aionclient.WithPartitionKey(key))
	if err != nil {
		t.Fatalf("Publish() error: %v", err)
	}
	if ack.Partition != wantPartition {
		t.Fatalf("PubAck.Partition = %d, want %d", ack.Partition, wantPartition)
	}
}

func TestFetchReturnsPartitionIntegration(t *testing.T) {
	nc := newTestClient(t)
	ctx := context.Background()

	if err := nc.CreateStream(ctx, aionclient.StreamConfig{
		Name:     "FETCH_PART",
		Subjects: []string{"fetch.>"},
	}); err != nil {
		t.Fatalf("CreateStream() error: %v", err)
	}

	if _, err := nc.Publish(ctx, "FETCH_PART", "fetch.event", []byte("payload")); err != nil {
		t.Fatalf("Publish() error: %v", err)
	}

	sub, err := nc.PullSubscribe(ctx, aionclient.PullSubscribeConfig{
		Stream:   "FETCH_PART",
		Consumer: "worker",
	})
	if err != nil {
		t.Fatalf("PullSubscribe() error: %v", err)
	}

	msgs, err := sub.Fetch(1, aionclient.MaxWait(time.Second))
	if err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("Fetch() len = %d, want 1", len(msgs))
	}
	if got := msgs[0].Partition(); got != 0 {
		t.Fatalf("Msg.Partition() = %d, want 0", got)
	}
}

func TestPublishWithExplicitPartitionIntegration(t *testing.T) {
	nc := newTestClient(t)
	ctx := context.Background()

	const partitions = 4
	if err := nc.CreateStream(ctx, aionclient.StreamConfig{
		Name:       "EXPLICIT",
		Subjects:   []string{"tasks.>"},
		Partitions: partitions,
	}); err != nil {
		t.Fatalf("CreateStream() error: %v", err)
	}

	const wantPartition = 2
	ack, err := nc.Publish(ctx, "EXPLICIT", "tasks.run", []byte("job"), aionclient.WithPartition(wantPartition))
	if err != nil {
		t.Fatalf("Publish() error: %v", err)
	}
	if ack.Partition != wantPartition {
		t.Fatalf("PubAck.Partition = %d, want %d", ack.Partition, wantPartition)
	}
}

func TestPublishPartitionBackwardCompat(t *testing.T) {
	nc := newTestClient(t)
	ctx := context.Background()

	if err := nc.CreateStream(ctx, aionclient.StreamConfig{
		Name:     "LEGACY",
		Subjects: []string{"legacy.>"},
	}); err != nil {
		t.Fatalf("CreateStream() error: %v", err)
	}

	ack, err := nc.Publish(ctx, "LEGACY", "legacy.event", []byte("x"))
	if err != nil {
		t.Fatalf("Publish() error: %v", err)
	}
	if ack.Partition != 0 {
		t.Fatalf("PubAck.Partition = %d, want 0", ack.Partition)
	}
}