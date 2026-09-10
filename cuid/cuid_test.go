package cuid_test

import (
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-client-sdk-go/cuid"
)

func TestNewOrgIDOpaqueCuid2(t *testing.T) {
	id := cuid.NewOrgID()
	if !strings.HasPrefix(id, cuid.OrgIDPrefix) {
		t.Fatalf("NewOrgID = %q, want %s prefix", id, cuid.OrgIDPrefix)
	}
	if !cuid.IsCuid(strings.TrimPrefix(id, cuid.OrgIDPrefix)) {
		t.Fatalf("NewOrgID suffix not cuid2: %q", id)
	}
	if !cuid.IsOpaqueOrgID(id) {
		t.Fatalf("IsOpaqueOrgID(%q) = false", id)
	}
	if cuid.IsOpaqueWorkspaceID(id) {
		t.Fatalf("IsOpaqueWorkspaceID(%q) = true", id)
	}
}

func TestNewWorkspaceIDOpaqueCuid2(t *testing.T) {
	id := cuid.NewWorkspaceID()
	if !strings.HasPrefix(id, cuid.WorkspaceIDPrefix) {
		t.Fatalf("NewWorkspaceID = %q, want %s prefix", id, cuid.WorkspaceIDPrefix)
	}
	if !cuid.IsCuid(strings.TrimPrefix(id, cuid.WorkspaceIDPrefix)) {
		t.Fatalf("NewWorkspaceID suffix not cuid2: %q", id)
	}
	if !cuid.IsOpaqueWorkspaceID(id) {
		t.Fatalf("IsOpaqueWorkspaceID(%q) = false", id)
	}
	if cuid.IsOpaqueOrgID(id) {
		t.Fatalf("IsOpaqueOrgID(%q) = true", id)
	}
}

func TestIsOpaqueOrgID(t *testing.T) {
	minted := cuid.NewOrgID()
	cases := []struct {
		id   string
		want bool
	}{
		{minted, true},
		{" " + minted + " ", true},
		{"org_example", false}, // local/dev placeholder — not CP-minted
		{"acme-org", false},
		{"org_", false},
		{"", false},
		{"   ", false},
		{cuid.NewWorkspaceID(), false},
	}
	for _, tc := range cases {
		if got := cuid.IsOpaqueOrgID(tc.id); got != tc.want {
			t.Errorf("IsOpaqueOrgID(%q) = %v, want %v", tc.id, got, tc.want)
		}
	}
}

func TestIsOpaqueWorkspaceID(t *testing.T) {
	minted := cuid.NewWorkspaceID()
	cases := []struct {
		id   string
		want bool
	}{
		{minted, true},
		{" " + minted + " ", true},
		{"ws_default", false}, // not a CP-minted id; omit-blank uses broker root-default
		{"ws_1", false},
		{"ws_", false},
		{"", false},
		{"   ", false},
		{cuid.NewOrgID(), false},
	}
	for _, tc := range cases {
		if got := cuid.IsOpaqueWorkspaceID(tc.id); got != tc.want {
			t.Errorf("IsOpaqueWorkspaceID(%q) = %v, want %v", tc.id, got, tc.want)
		}
	}
}
