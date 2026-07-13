// Package envelope defines the §11 agent message schema for the PoC simulation.
package envelope

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/iome-sh/iomesh-client-sdk-go/cuid"
)

const Version = 1

// Envelope is the JSON payload for agent simulation messages.
type Envelope struct {
	V             int             `json:"v"`
	Type          string          `json:"type"`
	AgentID       string          `json:"agent_id"`
	CorrelationID string          `json:"correlation_id"`
	CausationID   string          `json:"causation_id,omitempty"`
	Timestamp     string          `json:"timestamp"`
	Payload       json.RawMessage `json:"payload"`
	Metadata      Metadata        `json:"metadata"`
}

// Metadata carries optional routing and observability fields (§11).
type Metadata struct {
	DataProductID string   `json:"data_product_id,omitempty"`
	SchemaVersion string   `json:"schema_version,omitempty"`
	EmbeddingRef  *string  `json:"embedding_ref"`
	SurpriseScore *float64 `json:"surprise_score"`
	TraceID       string   `json:"trace_id,omitempty"`
}

// TaskPayload is the inner payload for type=task messages.
type TaskPayload struct {
	TaskType string `json:"task_type"`
	InputRef string `json:"input_ref"`
}

// ObservationPayload is the inner payload for type=observation messages.
type ObservationPayload struct {
	Summary  string `json:"summary"`
	InputRef string `json:"input_ref"`
}

// NewID returns a collision-resistant correlation identifier.
func NewID() (string, error) {
	return cuid.NewPrefixed("clk")
}

// Now returns an RFC3339 UTC timestamp.
func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// Marshal serializes an envelope to JSON.
func Marshal(e Envelope) ([]byte, error) {
	if e.V == 0 {
		e.V = Version
	}
	if e.Timestamp == "" {
		e.Timestamp = Now()
	}
	if e.Metadata.SchemaVersion == "" {
		e.Metadata.SchemaVersion = "1.0.0"
	}
	return json.Marshal(e)
}

// Unmarshal parses envelope JSON.
func Unmarshal(data []byte) (Envelope, error) {
	var e Envelope
	if err := json.Unmarshal(data, &e); err != nil {
		return Envelope{}, err
	}
	return e, nil
}

// UnmarshalPayload decodes the inner payload field into dst.
func UnmarshalPayload(raw json.RawMessage, dst any) error {
	return json.Unmarshal(raw, dst)
}

// NewTask builds a task envelope for the research summarization pipeline.
func NewTask(agentID, correlationID, inputRef string) ([]byte, error) {
	inner, err := json.Marshal(TaskPayload{
		TaskType: "summarize",
		InputRef: inputRef,
	})
	if err != nil {
		return nil, err
	}
	return Marshal(Envelope{
		Type:          "task",
		AgentID:       agentID,
		CorrelationID: correlationID,
		Payload:       inner,
		Metadata: Metadata{
			DataProductID: "research.tasks",
			EmbeddingRef:  nil,
			SurpriseScore: nil,
		},
	})
}

// NewAction builds an ephemeral action envelope (e.g. worker.processing).
func NewAction(agentID, correlationID, causationID string, inner any) ([]byte, error) {
	innerJSON, err := json.Marshal(inner)
	if err != nil {
		return nil, err
	}
	return Marshal(Envelope{
		Type:          "action",
		AgentID:       agentID,
		CorrelationID: correlationID,
		CausationID:   causationID,
		Payload:       innerJSON,
		Metadata: Metadata{
			DataProductID: "agent.events",
		},
	})
}

// NewResult builds an ephemeral result envelope (e.g. worker.completed).
func NewResult(agentID, correlationID, causationID string, inner any) ([]byte, error) {
	innerJSON, err := json.Marshal(inner)
	if err != nil {
		return nil, err
	}
	return Marshal(Envelope{
		Type:          "result",
		AgentID:       agentID,
		CorrelationID: correlationID,
		CausationID:   causationID,
		Payload:       innerJSON,
		Metadata: Metadata{
			DataProductID: "agent.events",
		},
	})
}

// NewObservation builds a durable observation envelope.
func NewObservation(agentID, correlationID, inputRef, summary string) ([]byte, error) {
	inner, err := json.Marshal(ObservationPayload{
		Summary:  summary,
		InputRef: inputRef,
	})
	if err != nil {
		return nil, err
	}
	return Marshal(Envelope{
		Type:          "observation",
		AgentID:       agentID,
		CorrelationID: correlationID,
		Payload:       inner,
		Metadata: Metadata{
			DataProductID: "research.findings",
		},
	})
}

// CheckpointSeq parses worker checkpoint values formatted as seq=N.
func CheckpointSeq(value []byte) (uint64, error) {
	const prefix = "seq="
	if len(value) < len(prefix) || string(value[:len(prefix)]) != prefix {
		return 0, fmt.Errorf("invalid checkpoint %q", value)
	}
	var seq uint64
	if _, err := fmt.Sscanf(string(value[len(prefix):]), "%d", &seq); err != nil {
		return 0, fmt.Errorf("parse checkpoint %q: %w", value, err)
	}
	return seq, nil
}
