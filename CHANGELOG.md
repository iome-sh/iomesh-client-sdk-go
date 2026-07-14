# Changelog

All notable changes to `github.com/iome-sh/iomesh-client-sdk-go` are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- **Primary package:** [`iomeshclient`](./iomeshclient) (public product naming).
- **Wire headers:** `X-IOMesh-Tenant`, `X-IOMesh-Org` (and related `X-IOMesh-*`).
- Kafka client id: `iomesh-kafka-client`.
- Examples: `IOMESH_*` env vars only.
- Docs: public naming policy for packages, env, and headers.

### Removed

- Package `aionclient` (no compatibility shim; use `iomeshclient`).
- Legacy `X-Aion-*` wire header names and `AION_*` example env aliases.

## [0.1.2] — 2026-07-13

### Changed

- Public docs: remove private platform repo links; copyright **IOMesh Technology Ltd.**
- Drop ignored integration tests that referenced private platform packages (keep pure-client unit tests).

### Fixed

- CI: gofmt, go 1.23/1.24 matrix, govulncheck on library packages with stable toolchain.

## [0.1.0] — 2026-07-13

### Added

- Initial public release of the I/O Mesh Go client SDK.
- Packages: `aionclient` (later renamed to `iomeshclient`), `connectorsdk`, `kafka`, `envelope`, `cuid`.
- CI (test, race, vet, govulncheck, golangci-lint), Dependabot, SECURITY.md, CONTRIBUTING.

### Notes

- Module path: `github.com/iome-sh/iomesh-client-sdk-go`.
