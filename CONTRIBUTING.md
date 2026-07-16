# Contributing

Thanks for improving the I/O Mesh Go client SDK.

**Copyright:** © 2026 IOMesh Technology Ltd.

## Ground rules

1. **Public API stability** — prefer additive changes; breaking changes require a major version and CHANGELOG entry.  
2. **Pure client only** — this repository must not depend on unpublished platform packages. Keep the module free of local `replace` directives before release.  
3. **Tests required** — unit tests for pure logic; use `httptest` for HTTP client behavior. No live broker credentials in CI.  
4. **Security** — follow [SECURITY.md](SECURITY.md); no credentials in fixtures.  

## Workflow

```bash
git clone https://github.com/iome-sh/iomesh-client-sdk-go.git
cd iomesh-client-sdk-go
go test ./...
go test -race ./...
```

Optional (if installed):

```bash
golangci-lint run ./...
```

1. Open a PR against `main` from a feature branch.  
2. Ensure CI is green (tests + lint + govulncheck).  
3. Keep commits focused; squash merge is preferred.  
4. Update [CHANGELOG.md](CHANGELOG.md) **Unreleased** for user-visible changes.  

## Code style

- Standard `gofmt` / `goimports`  
- Exported symbols need GoDoc comments  
- Prefer small packages with clear boundaries  

## Local development tip

When iterating from another local checkout of this module:

```go
// in your app's go.mod
replace github.com/iome-sh/iomesh-client-sdk-go => ../iomesh-client-sdk-go
```

Remove the `replace` before release; consumers should pin tagged versions only.

## Issues

Use [issue templates](https://github.com/iome-sh/iomesh-client-sdk-go/issues/new/choose). Support channels: [SUPPORT.md](SUPPORT.md).

## License

By contributing, you agree that your contributions are licensed under the MIT License (see [LICENSE](LICENSE)).

## Release process

See [RELEASING.md](RELEASING.md).
