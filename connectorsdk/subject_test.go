package connectorsdk

import (
	"strings"
	"testing"
)

func TestSubjectForDepartment(t *testing.T) {
	tests := []struct {
		name       string
		department string
		source     string
		want       string
		wantErr    bool
	}{
		{
			name:       "engineering slack",
			department: "engineering",
			source:     "slack",
			want:       "dept.engineering.events.slack",
		},
		{
			name:       "ops github",
			department: "ops",
			source:     "github",
			want:       "dept.ops.events.github",
		},
		{
			name:       "trims whitespace",
			department: "  support ",
			source:     " zendesk ",
			want:       "dept.support.events.zendesk",
		},
		{
			name:       "missing department",
			department: "",
			source:     "slack",
			wantErr:    true,
		},
		{
			name:       "missing source",
			department: "engineering",
			source:     "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SubjectForDepartment(tt.department, tt.source)
			if tt.wantErr {
				if err == nil {
					t.Fatal("SubjectForDepartment() = nil error, want error")
				}
				if !strings.Contains(err.Error(), "connectorsdk:") {
					t.Fatalf("error = %v, want connectorsdk prefix", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("SubjectForDepartment() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("SubjectForDepartment() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSubjectForWarehouse(t *testing.T) {
	tests := []struct {
		name       string
		department string
		source     string
		want       string
		wantErr    bool
	}{
		{
			name:       "finance snowflake",
			department: "finance",
			source:     "snowflake",
			want:       "dept.finance.views.warehouse.snowflake",
		},
		{
			name:       "sales bigquery",
			department: "sales",
			source:     "bigquery",
			want:       "dept.sales.views.warehouse.bigquery",
		},
		{
			name:       "missing department",
			department: "",
			source:     "snowflake",
			wantErr:    true,
		},
		{
			name:       "missing source",
			department: "finance",
			source:     "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SubjectForWarehouse(tt.department, tt.source)
			if tt.wantErr {
				if err == nil {
					t.Fatal("SubjectForWarehouse() = nil error, want error")
				}
				if !strings.Contains(err.Error(), "connectorsdk:") {
					t.Fatalf("error = %v, want connectorsdk prefix", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("SubjectForWarehouse() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("SubjectForWarehouse() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSubjectForMetric(t *testing.T) {
	tests := []struct {
		name       string
		department string
		source     string
		want       string
		wantErr    bool
	}{
		{
			name:       "finance dbt",
			department: "finance",
			source:     "dbt",
			want:       "dept.finance.views.metrics.dbt",
		},
		{
			name:       "sales dbt",
			department: "sales",
			source:     "dbt",
			want:       "dept.sales.views.metrics.dbt",
		},
		{
			name:       "missing department",
			department: "",
			source:     "dbt",
			wantErr:    true,
		},
		{
			name:       "missing source",
			department: "finance",
			source:     "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SubjectForMetric(tt.department, tt.source)
			if tt.wantErr {
				if err == nil {
					t.Fatal("SubjectForMetric() = nil error, want error")
				}
				if !strings.Contains(err.Error(), "connectorsdk:") {
					t.Fatalf("error = %v, want connectorsdk prefix", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("SubjectForMetric() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("SubjectForMetric() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSubjectForEmbedding(t *testing.T) {
	tests := []struct {
		name       string
		department string
		source     string
		want       string
		wantErr    bool
	}{
		{
			name:       "engineering memory",
			department: "engineering",
			source:     "memory",
			want:       "dept.engineering.events.embeddings.memory",
		},
		{
			name:       "product memory",
			department: "product",
			source:     "memory",
			want:       "dept.product.events.embeddings.memory",
		},
		{
			name:       "missing department",
			department: "",
			source:     "memory",
			wantErr:    true,
		},
		{
			name:       "missing source",
			department: "engineering",
			source:     "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SubjectForEmbedding(tt.department, tt.source)
			if tt.wantErr {
				if err == nil {
					t.Fatal("SubjectForEmbedding() = nil error, want error")
				}
				if !strings.Contains(err.Error(), "connectorsdk:") {
					t.Fatalf("error = %v, want connectorsdk prefix", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("SubjectForEmbedding() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("SubjectForEmbedding() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSubjectForDocument(t *testing.T) {
	tests := []struct {
		name       string
		department string
		source     string
		want       string
		wantErr    bool
	}{
		{
			name:       "product notion",
			department: "product",
			source:     "notion",
			want:       "dept.product.events.docs.notion",
		},
		{
			name:       "engineering confluence",
			department: "engineering",
			source:     "confluence",
			want:       "dept.engineering.events.docs.confluence",
		},
		{
			name:       "trims whitespace",
			department: " legal ",
			source:     " google_drive ",
			want:       "dept.legal.events.docs.google_drive",
		},
		{
			name:       "missing department",
			department: "",
			source:     "notion",
			wantErr:    true,
		},
		{
			name:       "missing source",
			department: "product",
			source:     "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SubjectForDocument(tt.department, tt.source)
			if tt.wantErr {
				if err == nil {
					t.Fatal("SubjectForDocument() = nil error, want error")
				}
				if !strings.Contains(err.Error(), "connectorsdk:") {
					t.Fatalf("error = %v, want connectorsdk prefix", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("SubjectForDocument() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("SubjectForDocument() = %q, want %q", got, tt.want)
			}
		})
	}
}
