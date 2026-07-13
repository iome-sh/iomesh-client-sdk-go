//go:build ignore

package aionclient_test

import (
	"context"
	"testing"

	"github.com/iome-sh/aion/pkg/aionclient"
)

func TestCreateSubset(t *testing.T) {
	nc := newTestClient(t)
	ctx := context.Background()

	if err := nc.CreateStream(ctx, aionclient.StreamConfig{
		Name:     "source",
		Subjects: []string{"dept.research.>"},
	}); err != nil {
		t.Fatalf("CreateStream() error: %v", err)
	}

	cfg := aionclient.SubsetConfig{
		ID:            "research-view",
		SourceStream:  "source",
		SubjectFilter: "dept.research.>",
		FieldMask:     []string{"ssn"},
		Tenant:        "dept.research",
	}
	if err := nc.CreateSubset(ctx, cfg); err != nil {
		t.Fatalf("CreateSubset() error: %v", err)
	}
}

func TestCreateSubsetValidation(t *testing.T) {
	nc := newTestClient(t)
	ctx := context.Background()

	if err := nc.CreateSubset(ctx, aionclient.SubsetConfig{}); err == nil {
		t.Fatal("CreateSubset() error = nil, want validation error")
	}
	if err := nc.CreateSubset(ctx, aionclient.SubsetConfig{ID: "only-id"}); err == nil {
		t.Fatal("CreateSubset() error = nil, want validation error")
	}
}
