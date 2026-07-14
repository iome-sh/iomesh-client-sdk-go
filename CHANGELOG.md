# Changelog

All notable changes to `github.com/iome-sh/iomesh-client-sdk-go` are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- **Public naming:** primary HTTP client package is [`iomeshclient`](./iomeshclient) (not internal codename).
- Kafka client id string: `iomesh-kafka-client`.
- Examples: prefer `IOMESH_*` env vars (`AION_*` temporary fallback only).
- Docs: public SDK naming policy (`iomesh` / `IOMESH_*`); wire headers `X-Aion-*` remain broker protocol.

### Deprecated

- Package [`aionclient`](./aionclient) — thin re-export of `iomeshclient` for import compatibility. Prefer `iomeshclient`.

### Notes

- Wire headers `X-Aion-Tenant` / `X-Aion-Org` are unchanged (broker contract).

## [0.1.2] — 2026-07-13

### Changed

- Public docs: remove private platform repo links; copyright **IOMesh Technology Ltd.**
- Drop ignored integration tests that referenced private platform packages (keep pure-client unit tests).

### Fixed

- CI: gofmt, go 1.23/1.24 matrix, govulncheck on library packages with stable toolchain.

## [0.1.0] — 2026-07-13

### Added

- Initial public release of the I/O Mesh Go client SDK.
- Packages: `aionclient`, `connectorsdk`, `kafka`, `envelope`, `cuid`.
- CI (test, race, vet, govulncheck, golangci-lint), Dependabot, SECURITY.md, CONTRIBUTING.

### Notes

- Module path: `github.com/iome-sh/iomesh-client-sdk-go`.
