// Package cuid wraps github.com/nrednav/cuid2 for consistent ID generation across I/O Mesh.
//
// Default IDs use cuid2.Generate() (length 24). Custom lengths use cuid2.Init()
// with cuid2.WithLength, matching https://github.com/nrednav/cuid2.
package cuid

import (
	"fmt"
	"sync"

	"github.com/nrednav/cuid2"
)

const (
	DefaultLength = cuid2.DefaultIdLength
	MinLength     = cuid2.MinIdLength
	MaxLength     = cuid2.MaxIdLength

	// SlugLength matches tools/cuid SHOW_SLUG (init with WithLength per library docs).
	SlugLength = 8
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
