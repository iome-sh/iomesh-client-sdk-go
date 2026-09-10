// Package cuid wraps github.com/nrednav/cuid2 for consistent ID generation across I/O Mesh.
//
// Default IDs use cuid2.Generate() (length 24). Custom lengths use cuid2.Init()
// with cuid2.WithLength, matching https://github.com/nrednav/cuid2.
//
// NewOrgID / NewWorkspaceID mint opaque public-id shapes (org_/ws_ + cuid2)
// matching X-IOMesh-Org / X-IOMesh-Workspace. IsOpaqueOrgID / IsOpaqueWorkspaceID
// validate those shapes (name slugs and placeholders are not opaque).
package cuid

import (
	"fmt"
	"strings"
	"sync"

	"github.com/nrednav/cuid2"
)

const (
	DefaultLength = cuid2.DefaultIdLength
	MinLength     = cuid2.MinIdLength
	MaxLength     = cuid2.MaxIdLength

	// SlugLength matches tools/cuid SHOW_SLUG (init with WithLength per library docs).
	SlugLength = 8

	// OrgIDPrefix is the control-plane public org id prefix (X-IOMesh-Org).
	OrgIDPrefix = "org_"
	// WorkspaceIDPrefix is the control-plane public workspace id prefix (X-IOMesh-Workspace).
	WorkspaceIDPrefix = "ws_"
)

var (
	slugOnce sync.Once
	slugGen  func() string
	slugErr  error
)

func initSlug() (func() string, error) {
	slugOnce.Do(func() {
		slugGen, slugErr = cuid2.Init(cuid2.WithLength(SlugLength))
	})
	return slugGen, slugErr
}

// New returns a collision-resistant identifier using cuid2.Generate().
func New() (string, error) {
	return cuid2.Generate(), nil
}

// MustNew is like New but panics only if used with a future error-returning path.
func MustNew() string {
	return cuid2.Generate()
}

// NewPrefixed returns prefix + New().
func NewPrefixed(prefix string) (string, error) {
	return prefix + cuid2.Generate(), nil
}

// MustNewPrefixed is like NewPrefixed without error handling.
func MustNewPrefixed(prefix string) string {
	return prefix + cuid2.Generate()
}

// NewSlug returns a short identifier via cuid2.Init(cuid2.WithLength(SlugLength)).
func NewSlug() (string, error) {
	gen, err := initSlug()
	if err != nil {
		return "", fmt.Errorf("cuid: init slug generator: %w", err)
	}
	return gen(), nil
}

// MustNewSlug is like NewSlug but panics on init failure.
func MustNewSlug() string {
	id, err := NewSlug()
	if err != nil {
		panic(err)
	}
	return id
}

// IsCuid reports whether id matches cuid2's validation rules.
func IsCuid(id string) bool {
	return cuid2.IsCuid(id)
}

// NewOrgID mints an opaque organization tracking id: org_ + cuid2.
// Shape matches the control-plane public id used in X-IOMesh-Org.
// Hosted brokers still issue the authoritative org id; this helper does not register an organization.
func NewOrgID() string {
	return MustNewPrefixed(OrgIDPrefix)
}

// NewWorkspaceID mints an opaque workspace tracking id: ws_ + cuid2.
// Shape matches the control-plane public id used in X-IOMesh-Workspace.
// Hosted brokers still issue the authoritative workspace id; this helper does not register a workspace.
func NewWorkspaceID() string {
	return MustNewPrefixed(WorkspaceIDPrefix)
}

// IsOpaqueOrgID reports whether id is org_ + default-length cuid2
// (the shape NewOrgID / control-plane mint produce). Name slugs and
// placeholders such as org_example are not opaque.
func IsOpaqueOrgID(id string) bool {
	return isOpaquePrefixedID(id, OrgIDPrefix)
}

// IsOpaqueWorkspaceID reports whether id is ws_ + default-length cuid2
// (the shape NewWorkspaceID / control-plane mint produce). Name slugs and
// placeholders such as ws_default are not opaque.
func IsOpaqueWorkspaceID(id string) bool {
	return isOpaquePrefixedID(id, WorkspaceIDPrefix)
}

func isOpaquePrefixedID(id, prefix string) bool {
	id = strings.TrimSpace(id)
	if !strings.HasPrefix(id, prefix) {
		return false
	}
	suffix := strings.TrimPrefix(id, prefix)
	// Generate() / NewPrefixed mint DefaultLength (24). cuid2.IsCuid alone
	// accepts 2..32, which would treat org_example / ws_default as opaque.
	return len(suffix) == DefaultLength && IsCuid(suffix)
}
