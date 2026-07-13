package connectorsdk

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iome-sh/iomesh-client-sdk-go/envelope"
)

// ConnectorPayload is the inner observation payload for generic partner events.
type ConnectorPayload struct {
	ConnectorID string          `json:"connector_id"`
	Source      string          `json:"source"`
	Department  string          `json:"department"`
	ExternalID  string          `json:"external_id"`
	EventType   string          `json:"event_type,omitempty"`
	Raw         json.RawMessage `json:"raw,omitempty"`
}

// NormalizeEnvelope builds an internal observation envelope from verified inbound
// partner webhook fields (slack/github normalize pattern).
func NormalizeEnvelope(connectorID, department, source, externalID, eventType string, event json.RawMessage) ([]byte, error) {
	connectorID = strings.TrimSpace(connectorID)
	if connectorID == "" {
		return nil, fmt.Errorf("connectorsdk: connector id required")
	}
	dept := strings.TrimSpace(department)
	if dept == "" {
		return nil, fmt.Errorf("connectorsdk: department required")
	}
	src := strings.TrimSpace(source)
	if src == "" {
		return nil, fmt.Errorf("connectorsdk: source required")
	}

	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		id, err := envelope.NewID()
		if err != nil {
			return nil, fmt.Errorf("connectorsdk: correlation id: %w", err)
		}
		externalID = id
	}

	subject, err := SubjectForDepartment(dept, src)
	if err != nil {
		return nil, err
	}

	inner, err := json.Marshal(ConnectorPayload{
		ConnectorID: connectorID,
		Source:      src,
		Department:  dept,
		ExternalID:  externalID,
		EventType:   strings.TrimSpace(eventType),
		Raw:         event,
	})
	if err != nil {
		return nil, fmt.Errorf("connectorsdk: marshal connector payload: %w", err)
	}

	return envelope.Marshal(envelope.Envelope{
		Type:          "observation",
		AgentID:       "connector:" + connectorID,
		CorrelationID: externalID,
		Payload:       inner,
		Metadata: envelope.Metadata{
			DataProductID: subject,
		},
	})
}

// PublishHeaders returns broker metadata for connector ingress (v10 pattern).
func PublishHeaders(connectorID, department, externalID, source string) map[string]string {
	return map[string]string{
		"connector_id": strings.TrimSpace(connectorID),
		"department":   strings.TrimSpace(department),
		"external_id":  strings.TrimSpace(externalID),
		"source":       strings.TrimSpace(source),
	}
}