package aionclient

import (
	"context"
	"errors"
	"strings"
)

// MeshSDKConfig wires HTTP (NATS-style) and optional Kafka protocol endpoints.
type MeshSDKConfig struct {
	BrokerURL string
	KafkaAddr string
	Tenant    string
}

// MeshSDK is a unified facade over pkg/aionclient HTTP and Kafka protocol produce.
type MeshSDK struct {
	http  *Client
	kafka *KafkaClient
}

// NewMeshSDK returns a mesh client facade. No network I/O is performed at connect time.
func NewMeshSDK(cfg MeshSDKConfig, opts ...ConnectOpt) (*MeshSDK, error) {
	brokerURL := strings.TrimSpace(cfg.BrokerURL)
	if brokerURL == "" {
		return nil, errors.New("aionclient: BrokerURL required")
	}

	connectOpts := append([]ConnectOpt(nil), opts...)
	if tenant := strings.TrimSpace(cfg.Tenant); tenant != "" {
		connectOpts = append(connectOpts, WithTenant(tenant))
	}

	httpClient, err := Connect(Options{URL: brokerURL}, connectOpts...)
	if err != nil {
		return nil, err
	}

	var kafka *KafkaClient
	if addr := strings.TrimSpace(cfg.KafkaAddr); addr != "" {
		kafka = NewKafkaClient(addr)
	}

	return &MeshSDK{http: httpClient, kafka: kafka}, nil
}

// Pub publishes an ephemeral fire-and-forget message via POST /v1/pub.
func (m *MeshSDK) Pub(ctx context.Context, subject string, payload []byte, headers map[string]string) error {
	if m == nil || m.http == nil {
		return errors.New("aionclient: mesh sdk http client not configured")
	}
	return m.http.Pub(ctx, subject, payload, headers)
}

// Publish appends a message to stream via POST /v1/streams/{stream}/publish.
func (m *MeshSDK) Publish(
	ctx context.Context, stream, subject string, payload []byte, opts ...PublishOpt,
) (*PubAck, error) {
	if m == nil || m.http == nil {
		return nil, errors.New("aionclient: mesh sdk http client not configured")
	}
	return m.http.Publish(ctx, stream, subject, payload, opts...)
}

// EnsureStream creates the stream if it does not already exist.
func (m *MeshSDK) EnsureStream(ctx context.Context, cfg StreamConfig) error {
	if m == nil || m.http == nil {
		return errors.New("aionclient: mesh sdk http client not configured")
	}
	return m.http.EnsureStream(ctx, cfg)
}

// KafkaProduce sends one Kafka protocol record when KafkaAddr was configured.
func (m *MeshSDK) KafkaProduce(
	ctx context.Context, topic string, partition int32, key, value []byte,
) (int64, error) {
	if m == nil || m.kafka == nil {
		return 0, errors.New("aionclient: KafkaAddr not configured")
	}
	return m.kafka.Produce(ctx, topic, partition, key, value)
}

// Close closes the optional Kafka connection.
func (m *MeshSDK) Close() error {
	if m == nil || m.kafka == nil {
		return nil
	}
	return m.kafka.Close()
}
