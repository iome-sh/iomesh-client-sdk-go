# Changelog

All notable changes to `github.com/iome-sh/iomesh-client-sdk-go` are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **`Health` / `Ready`** — `GET /health` and `GET /ready` then `/readyz` (iomesh-tui dogfood parity); sends User-Agent
- Example `memory-metering-dogfood` probes health/ready first

## [0.6.0] — 2026-07-19

Minor release: public Version constant + default User-Agent on all HTTP. Compatible with `v0.5.x`.

### Added

- **`Version` / User-Agent** — default `User-Agent: iomesh-client-sdk-go/<Version>` on all HTTP; `WithUserAgent` override; public `iomeshclient.Version` constant (s287)

## [0.5.0] — 2026-07-19

Minor release: DualWriteMemoryTurn helper (async stream + optional fail-open sync). Compatible with `v0.4.x`.

### Added

- **`DualWriteMemoryTurn`** — async `MEMORY_INGEST` plus optional fail-open sync `IngestMemoryTurn` (iomesh-tui dual_write semantics; optional SyncClient for sidecar URL)
- README dual-write usage notes

## [0.4.0] — 2026-07-19

Minor release: dept metering emit helpers + stage dogfood example. Compatible with `v0.3.x`.

### Added

- **`EmitDeptEvent` / `EmitLLMCall`** — publish `dept.*` / `dept.agent.llm_call` via `POST /v1/streams/dept/publish` (base64 envelope; multi-tenant headers + payload org/workspace; parity with iomesh-tui remote metering)
- **Example** `examples/memory-metering-dogfood` — stage smoke for dual-write ingest, session recall, sync retrieve, llm_call emit

## [0.3.0] — 2026-07-18

Minor release: aion/iomesh-tui memory + multi-tenant header parity. Compatible with `v0.2.x` callers (`RequestMemoryRecall` signature unchanged).

### Added

- Open-source process pack: SUPPORT, NOTICE, issue/PR templates, OPEN_SOURCE_AUDIT.
- `Connect` rejects non-`http(s)` broker URLs and URL-embedded userinfo (use `WithBearerToken`).
- **M3** temporal fields on `MemoryEnvelope` (`event_time`, `session_seq`, `entity_refs`, …) + `MemoryEntityRef` (backward-compatible omitempty).
- **M2** sync HTTP memory APIs: `RetrieveMemory` / `IngestMemoryTurn` (async `RequestMemoryRecall` / `PublishMemoryIngest` retained).
- **`WithWorkspace`** — sets `X-IOMesh-Workspace` on all requests (PlanGate / multi-tenant metering; parity with iomesh-tui).
- **`RequestMemoryRecallFull`** + `MemoryRecallRequest` — optional `session_id` on async MEMORY_RPC (TUI dogfood correlation).
- **RetrieveMemory path fallback** — tries `POST /v1/memory/retrieve` then `/v5`; sets `Path` on success; allows `session_id`-only queries.
- **IngestMemoryTurn path fallback** — tries `/v1/memory/ingest` then `/v5`.

### Changed

- SECURITY.md: GitHub private advisory path + residual risk table; supported versions include `v0.3.x`.
- Org branding: README footer Maintained-by line + NOTICE website; GitHub About homepage → [iome.sh](https://iome.sh).
- RELEASING.md: when to bump/tag (aligned with public iomesh-tui process).

### Previously (naming cleanup, shipped in v0.2.0)

- **Primary package:** [`iomeshclient`](./iomeshclient) (public product naming).
- **Wire headers:** `X-IOMesh-Tenant`, `X-IOMesh-Org` (and related `X-IOMesh-*`).
- Kafka client id: `iomesh-kafka-client`.
- Examples: `IOMESH_*` env vars only.
- Docs: public naming policy for packages, env, and headers.
- Removed package `aionclient` and legacy `X-Aion-*` / `AION_*` aliases.

[Unreleased]: https://github.com/iome-sh/iomesh-client-sdk-go/compare/v0.6.0...HEAD
[0.6.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.6.0
[0.5.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.5.0
[0.4.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.4.0
[0.3.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.3.0
[0.2.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.2.0
