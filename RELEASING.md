# Releasing

## Prerequisites

- [ ] `go test ./...` and `go test -race ./...` green  
- [ ] CHANGELOG.md updated under `## [Unreleased]` → version section  
- [ ] No accidental monorepo `replace` directives in `go.mod`  
- [ ] Maintainer rights on `iome-sh/iomesh-client-sdk-go`  

## Steps

```bash
# 1. Ensure main is clean and CI green
git checkout main && git pull --ff-only

# 2. Tag (annotated)
VERSION=v0.1.0
git tag -a "$VERSION" -m "iomesh-client-sdk-go $VERSION"
git push origin main "$VERSION"

# 3. Consumers
# go get github.com/iome-sh/iomesh-client-sdk-go@$VERSION
```

GitHub Actions will build/test on the tag. Publish is **module proxy only** (`proxy.golang.org` picks up the tag); no separate binary release required unless noted in CHANGELOG.

## Version policy

| Change | Version bump |
|--------|----------------|
| Bugfix, docs | PATCH |
| New APIs, backward compatible | MINOR |
| Breaking API / behavior | MAJOR |

## Deprecations

Mark with GoDoc `// Deprecated:` for at least one minor release before removal in a major.
