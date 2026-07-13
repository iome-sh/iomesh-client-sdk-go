package connectorsdk

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-client-sdk-go/envelope"
)

func TestNormalizeEnvelope(t *testing.T) {
	event := json.RawMessage(`{"type":"message","text":"hello"}`)

	tests := []struct {
		name        string
		connectorID string
		department  string
		source      string
		externalID  string
		eventType   string
		event       json.RawMessage
		wantErr     bool
		errContains string
		check       func(t *testing.T, env envelope.Envelope, inner ConnectorPayload)
	}{
		{
			name:        "valid with external id",
			connectorID: "slack",
			department:  "engineering",
			source:      "slack",
			externalID:  "Ev001",
			eventType:   "message",
			event:       event,
			check: func(t *testing.T, env envelope.Envelope, inner ConnectorPayload) {
				t.Helper()
				if env.Type != "observation" {
					t.Fatalf("type = %q, want observation", env.Type)
				}
				if env.AgentID != "connector:slack" {
					t.Fatalf("agent_id = %q, want connector:slack", env.AgentID)
				}
				if env.CorrelationID != "Ev001" {
					t.Fatalf("correlation_id = %q, want Ev001", env.CorrelationID)
				}
				if env.Metadata.DataProductID != "dept.engineering.events.slack" {
					t.Fatalf("data_product_id = %q, want dept.engineering.events.slack", env.Metadata.DataProductID)
				}
				if inner.Source != "slack" || inner.Department != "engineering" || inner.ExternalID != "Ev001" {
					t.Fatalf("inner = %+v, want slack/engineering/Ev001", inner)
				}
				if inner.EventType != "message" {
					t.Fatalf("event_type = %q, want message", inner.EventType)
				}
			},
		},
		{
			name:        "generates external id when empty",
			connectorID: "github",
			department:  "ops",
			source:      "github",
			externalID:  "",
			eventType:   "push",
			event:       json.RawMessage(`{"ref":"refs/heads/main"}`),
			check: func(t *testing.T, env envelope.Envelope, inner ConnectorPayload) {
				t.Helper()
				if env.CorrelationID == "" {
					t.Fatal("correlation_id empty, want generated id")
				}
				if !strings.HasPrefix(env.CorrelationID, "clk") {
					t.Fatalf("correlation_id = %q, want clk prefix", env.CorrelationID)
				}
				if inner.ExternalID != env.CorrelationID {
					t.Fatalf("inner.external_id = %q, want %q", inner.ExternalID, env.CorrelationID)
				}
			},
		},
		{
			name:        "missing connector id",
			connectorID: "",
			department:  "engineering",
			source:      "slack",
			wantErr:     true,
			errContains: "connector id required",
		},
		{
			name:        "missing department",
			connectorID: "slack",
			department:  "",
			source:      "slack",
			wantErr:     true,
			errContains: "department required",
		},
		{
			name:        "missing source",
			connectorID: "slack",
			department:  "engineering",
			source:      "",
			wantErr:     true,
			errContains: "source required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := NormalizeEnvelope(
				tt.connectorID,
				tt.department,
				tt.source,
				tt.externalID,
				tt.eventType,
				tt.event,
			)
			if tt.wantErr {
				if err == nil {
					t.Fatal("NormalizeEnvelope() = nil error, want error")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("error = %v, want contains %q", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeEnvelope() error = %v", err)
			}

			env, err := envelope.Unmarshal(payload)
			if err != nil {
				t.Fatalf("Unmarshal envelope: %v", err)
			}

			var inner ConnectorPayload
			if err := envelope.UnmarshalPayload(env.Payload, &inner); err != nil {
				t.Fatalf("UnmarshalPayload: %v", err)
			}

			if tt.check != nil {
				tt.check(t, env, inner)
			}
		})
	}
}

func TestPublishHeaders(t *testing.T) {
	tests := []struct {
		name        string
		connectorID string
		department  string
		externalID  string
		source      string
		want        map[string]string
	}{
		{
			name:        "github ingress",
			connectorID: "github",
			department:  "ops",
			externalID:  "delivery-002",
			source:      "github",
			want: map[string]string{
				"connector_id": "github",
				"department":   "ops",
				"external_id":  "delivery-002",
				"source":       "github",
			},
		},
		{
			name:        "trims whitespace",
			connectorID: " slack ",
			department:  " engineering ",
			externalID:  " Ev003 ",
			source:      " slack ",
			want: map[string]string{
				"connector_id": "slack",
				"department":   "engineering",
				"external_id":  "Ev003",
				"source":       "slack",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PublishHeaders(tt.connectorID, tt.department, tt.externalID, tt.source)
			for k, want := range tt.want {
				if got[k] != want {
					t.Fatalf("headers[%q] = %q, want %q (full = %#v)", k, got[k], want, got)
				}
			}
		})
	}
}
