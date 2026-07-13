package connectorsdk

import (
	"fmt"
	"strings"
)

// SubjectForDepartment returns the broker ingest subject for a department and source,
// e.g. dept.engineering.events.slack.
func SubjectForDepartment(department, source string) (string, error) {
	dept := strings.TrimSpace(department)
	if dept == "" {
		return "", fmt.Errorf("connectorsdk: department required")
	}
	src := strings.TrimSpace(source)
	if src == "" {
		return "", fmt.Errorf("connectorsdk: source required")
	}
	return fmt.Sprintf("dept.%s.events.%s", dept, src), nil
}

// SubjectForDocument returns the broker ingest subject for knowledge-mesh document
// events, e.g. dept.product.events.docs.notion. See docs/connectors/knowledge-mesh.md.
func SubjectForDocument(department, source string) (string, error) {
	dept := strings.TrimSpace(department)
	if dept == "" {
		return "", fmt.Errorf("connectorsdk: department required")
	}
	src := strings.TrimSpace(source)
	if src == "" {
		return "", fmt.Errorf("connectorsdk: source required")
	}
	return fmt.Sprintf("dept.%s.events.docs.%s", dept, src), nil
}

// SubjectForEmbedding returns the broker ingest subject for streaming embedding
// observations, e.g. dept.engineering.events.embeddings.memory.
func SubjectForEmbedding(department, source string) (string, error) {
	dept := strings.TrimSpace(department)
	if dept == "" {
		return "", fmt.Errorf("connectorsdk: department required")
	}
	src := strings.TrimSpace(source)
	if src == "" {
		return "", fmt.Errorf("connectorsdk: source required")
	}
	return fmt.Sprintf("dept.%s.events.embeddings.%s", dept, src), nil
}

// SubjectForWarehouse returns the broker view subject for analytical warehouse CDC
// products, e.g. dept.finance.views.warehouse.snowflake.
func SubjectForWarehouse(department, source string) (string, error) {
	dept := strings.TrimSpace(department)
	if dept == "" {
		return "", fmt.Errorf("connectorsdk: department required")
	}
	src := strings.TrimSpace(source)
	if src == "" {
		return "", fmt.Errorf("connectorsdk: source required")
	}
	return fmt.Sprintf("dept.%s.views.warehouse.%s", dept, src), nil
}

// SubjectForMetric returns the broker view subject for analytical metric products,
// e.g. dept.finance.views.metrics.dbt.
func SubjectForMetric(department, source string) (string, error) {
	dept := strings.TrimSpace(department)
	if dept == "" {
		return "", fmt.Errorf("connectorsdk: department required")
	}
	src := strings.TrimSpace(source)
	if src == "" {
		return "", fmt.Errorf("connectorsdk: source required")
	}
	return fmt.Sprintf("dept.%s.views.metrics.%s", dept, src), nil
}