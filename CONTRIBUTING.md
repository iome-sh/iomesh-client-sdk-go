# Contributing

Thanks for improving the I/O Mesh Go client SDK.

## Ground rules

1. **Public API stability** — prefer additive changes; breaking changes require a major version and CHANGELOG entry.  
2. **No monorepo secrets** — this repository must never import private `iome-sh/aion` packages.  
3. **Tests required** — unit tests for pure logic; use `httptest` for HTTP client behavior.  
4. **Security** — follow [SECURITY.md](SECURITY.md); no credentials in fixtures.  

## Workflow

```bash
git clone git@github.com:iome-sh/iomesh-client-sdk-go.git
cd iomesh-client-sdk-go
go test ./...
```

1. Open a PR against `main` from a feature branch.  
2. Ensure CI is green (tests + lint).  
3. Keep commits focused; squash merge is preferred.  

## Code style

- Standard `gofmt` / `goimports`  
- Exported symbols need GoDoc comments  
- Prefer small packages with clear boundaries  

## Local integration with the platform monorepo

While developing against a local `aion` checkout:

```go
// aion/go.mod
replace github.com/iome-sh/iomesh-client-sdk-go => ../iomesh-client-sdk-go
```

Remove the `replace` before release; consumers should pin tagged versions only.

## Release process

See [RELEASING.md](RELEASING.md).
