## Summary

<!-- What and why (1–3 bullets). -->

-

## Type of change

- [ ] Feature (additive API)
- [ ] Bug fix
- [ ] Security / hardening
- [ ] Docs / CI
- [ ] Breaking change (requires major version + CHANGELOG)

## Test plan

- [ ] `go test ./...` (and race if touching concurrency / HTTP)
- [ ] New/changed behavior covered by unit tests (`httptest` only — no live broker in CI)
- [ ] No secrets in examples/logs
- [ ] CHANGELOG **Unreleased** updated for user-visible changes

## Compatibility

- [ ] No accidental `replace` directives in `go.mod`
- [ ] Exported API changes documented (GoDoc + CHANGELOG)
