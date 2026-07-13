package aionclient

import (
	"context"
	"errors"
	"strings"

	kafkaproto "github.com/iome-sh/iomesh-client-sdk-go/kafka"
)

// KafkaClient speaks the Aion Kafka protocol subset (Produce) for mesh integrations.
type KafkaClient struct {
	inner *kafkaproto.Client
}

// NewKafkaClient returns a protocol client for addr (e.g. 127.0.0.1:9423).
func NewKafkaClient(addr string) *KafkaClient {
	return &KafkaClient{inner: kafkaproto.NewClient(strings.TrimSpace(addr))}
}

// Produce sends one record to topic/partition and returns the broker offset.
func (c *KafkaClient) Produce(
	ctx context.Context, topic string, partition int32, key, value []byte,
) (int64, error) {
	if c == nil || c.inner == nil {
		return 0, errors.New("aionclient: kafka client not configured")
	}
	return c.inner.Produce(ctx, topic, partition, key, value)
}

// Close closes any persistent Kafka connection.
func (c *KafkaClient) Close() error {
	if c == nil || c.inner == nil {
		return nil
	}
	return c.inner.Close()
}