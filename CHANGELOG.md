# Changelog

All notable changes to `github.com/iome-sh/iomesh-client-sdk-go` are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] — 2026-07-13

### Added

- Initial public release of the I/O Mesh Go client SDK.
- Packages: `aionclient`, `connectorsdk`, `kafka`, `envelope`, `cuid`.
- Migrated from monorepo path `github.com/iome-sh/mesh-client-sdk-go` (`aion/external/mesh-client-sdk-go`).
- CI (test, race, vet, govulncheck, golangci-lint), Dependabot, SECURITY.md, CONTRIBUTING.

### Notes

- Integration tests that require the platform monorepo harness remain `//go:build ignore` (run from `aion`).
- Module path changed: use `github.com/iome-sh/iomesh-client-sdk-go` for all new consumers.

## [0.1.2] — 2026-07-13

### Fixed

- CI: gofmt, go 1.23/1.24 matrix, govulncheck on library packages with stable toolchain.
