# Releasing

Public module: `github.com/iome-sh/iomesh-client-sdk-go`  
Consumers: `go get github.com/iome-sh/iomesh-client-sdk-go@vX.Y.Z` (module proxy).

## When to bump and tag

**Do not leave feature waves only under `[Unreleased]`.** After merging a coherent API surface (new methods, headers, wire parity with aion / iomesh-tui), cut a release in the same delivery loop:

| Trigger | Bump | Examples |
|---------|------|----------|
| New APIs / wire fields (backward compatible) | **minor** | M2 sync retrieve, `WithWorkspace`, session_id recall |
| Bug fix, docs-only, security residual wording | **patch** | URL validation fix, CHANGELOG typo |
| Breaking API / header renames | **major** | Removing `RequestMemoryRecall` 3-arg form |

Align with public **iomesh-tui** practice: ship tag when the wave is claim-ready (tests green + CHANGELOG + SECURITY supported-versions).

## Prerequisites

- [ ] `go test ./...` and `go test -race ./...` green  
- [ ] CI on the release commit green  
- [ ] CHANGELOG.md: `[Unreleased]` → `## [X.Y.Z]`  
- [ ] SECURITY.md supported-versions table lists latest minor  
- [ ] No accidental local `replace` directives in `go.mod`  
- [ ] Maintainer rights on `iome-sh/iomesh-client-sdk-go`  

## Steps

```bash
git checkout main && git pull --ff-only
# edit CHANGELOG + SECURITY supported versions
git commit -am "chore: release vX.Y.Z"
git push origin main

VERSION=vX.Y.Z
git tag -a "$VERSION" -m "iomesh-client-sdk-go $VERSION"
git push origin "$VERSION"

# Consumers
# go get github.com/iome-sh/iomesh-client-sdk-go@$VERSION
```

GitHub Actions run on the tag. Publish is **module proxy only** (`proxy.golang.org`); no binary artifacts unless noted in CHANGELOG.

## Version policy

| Change | Version bump |
|--------|----------------|
| Bugfix, docs | PATCH |
| New APIs, backward compatible | MINOR |
| Breaking API / behavior | MAJOR |

## Deprecations

Mark with GoDoc `// Deprecated:` for at least one minor release before removal in a major.

## Related public products

| Repo | Role |
|------|------|
| [iomesh-tui](https://github.com/iome-sh/iomesh-tui) | Agent harness (lean HTTP mirror of this SDK’s memory/mesh surfaces) |
| [iome.sh](https://iome.sh) | Platform product site |
