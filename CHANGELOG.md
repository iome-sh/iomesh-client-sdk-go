# Changelog

All notable changes to `github.com/iome-sh/iomesh-client-sdk-go` are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.17.0] — 2026-07-19

Minor release: FormatKV helpers and CreateBucket BucketInfo. Pre-1.0 CreateBucket signature change.

### Added

- **`FormatKVEntry` / `FormatKVKeys`** — pure operator helpers for KV entry detail and key listings (iomesh-tui CLI parity; no network I/O)
- **`BucketInfo`** — public bucket metadata type returned by `CreateBucket`

### Changed

- **`CreateBucket`** — now returns `(*BucketInfo, error)` instead of `error` only. On 201, decodes bucket metadata from the response body; on 409 conflict, success with `&BucketInfo{Name: name}` (name only). Empty name / nil client → error. **Breaking for pre-1.0 callers** that assigned a single return value — update to two return values

## [0.16.0] — 2026-07-19

Minor release: ListStreamMessages stream replay helper.

### Added

- **`ListStreamMessages`** — stream replay/read-range (`GET /v1/streams/{name}/messages`); `StreamMessage` + `ListStreamMessagesOptions` (`from_seq`/`to_seq`/`limit`); base64 payload decode with soft raw-bytes fallback; non-2xx → `*APIError`

## [0.15.0] — 2026-07-19

Minor release: stream format helpers and package examples; public-surface hygiene.

### Added

- **`FormatStreams` / `FormatStreamDetail`** — pure operator table/detail helpers for `StreamInfo` (iomesh-tui CLI parity; no network I/O)
- **Package examples** (`example_test.go`) — godoc Examples for format/diagnostics helpers (`FormatStreams`, `FormatStreamDetail`, `FormatConnectionStatus`, `FormatContextSnippet`, `PolicyDecision.Summary` / `ShouldBlockTool`)

### Changed

- **Public-surface hygiene** — CONTRIBUTING public repository policy; strip private continuum serials from CHANGELOG; replace private monorepo codenames in docs/comments with I/O Mesh broker/platform language (historical `aionclient` package rename retained as public API history)

## [0.14.0] — 2026-07-19

Minor release: CreateStream and EnsureStream return *StreamInfo (pre-1.0 signature change).

### Changed

- **`CreateStream` / `EnsureStream`** — now return `(*StreamInfo, error)` instead of `error` only. On 201, decodes stream metadata from the response body; on 409 conflict, best-effort `GetStream` (returns info if found, else `(nil, nil)` success without metadata). Same for `MeshSDK.EnsureStream`. **Breaking for pre-1.0 callers** that assigned a single return value — update to two return values

## [0.13.0] — 2026-07-19

Minor release: DeleteStream. Compatible with `v0.12.x`.

### Added

- **`DeleteStream`** — `DELETE /v1/streams/{name}` (204 success; empty name / nil client → error; non-2xx → `*APIError`)

## [0.12.0] — 2026-07-19

Minor release: ListStreams and GetStream. Compatible with `v0.11.x`.

### Added

- **`ListStreams` / `GetStream`** — explicit stream discovery (`GET /v1/streams`, `GET /v1/streams/{name}`); `StreamInfo` wire type; non-2xx returns `*APIError` (not fail-open empty); optional list envelope `{"streams":[...]}`

## [0.11.0] — 2026-07-19

Minor release: ConnectionStatus diagnostics helper. Compatible with `v0.10.x`.

### Added

- **`ConnectionStatus`** — fail-open one-shot diagnostics (`Health` + `Ready` + identity fields); `FormatConnectionStatus` / `FormatConnectionStatusJSON` for operators/CI (iomesh-tui mesh status parity)

## [0.10.0] — 2026-07-19

Minor release: QueryContext + ContextSnippet context plane. Compatible with `v0.9.x`.

### Added

- **`QueryContext` / `ContextSnippet`** — fail-open context plane (`POST /v1/context/query`); `LineageRef` / `ContextResult` / `QueryContextRequest`; `FormatContextSnippet` with `<iomesh-lineage>` (max 12 refs); agent `ContextSnippet` defaults `IncludeLineage=true` (iomesh-tui parity)

## [0.9.0] — 2026-07-19

Minor release: ListCatalog/GetCatalogProduct + WaitReady. Compatible with `v0.8.x`.

### Added

- **`ListCatalog` / `GetCatalogProduct`** — fail-open catalog plane (mesh `/v1/catalog/*` then portal `/v17|/v16` federation); multi-shape decode; `CatalogProduct` (+ Normalize); `FormatCatalog` / `FormatProductDetail` (iomesh-tui parity; named `CatalogProduct` to avoid clash with registry `DataProduct`)
- **`WaitReady`** — poll `Ready` (optional `RequireHealth`) until success or context done; default interval 500ms

## [0.8.0] — 2026-07-19

Minor release: EvaluatePolicy fail-open mesh policy helper. Compatible with `v0.7.x`.

### Added

- **`EvaluatePolicy`** — public fail-open mesh policy helper (`POST /v1/policy/evaluate`); per-call `PolicyMode` (`off`|`advisory`|`enforce`); `ShouldBlockTool` / `Summary` (iomesh-tui semantics without auto dept emit)

## [0.7.0] — 2026-07-19

Minor release: Health and Ready probes (TUI dogfood parity). Compatible with `v0.6.x`.

### Added

- **`Health` / `Ready`** — `GET /health` and `GET /ready` then `/readyz` (iomesh-tui dogfood parity); sends User-Agent
- Example `memory-metering-dogfood` probes health/ready first

## [0.6.0] — 2026-07-19

Minor release: public Version constant + default User-Agent on all HTTP. Compatible with `v0.5.x`.

### Added

- **`Version` / User-Agent** — default `User-Agent: iomesh-client-sdk-go/<Version>` on all HTTP; `WithUserAgent` override; public `iomeshclient.Version` constant

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

Minor release: I/O Mesh platform / iomesh-tui memory + multi-tenant header parity. Compatible with `v0.2.x` callers (`RequestMemoryRecall` signature unchanged).

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

[Unreleased]: https://github.com/iome-sh/iomesh-client-sdk-go/compare/v0.17.0...HEAD
[0.17.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.17.0
[0.16.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.16.0
[0.15.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.15.0
[0.14.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.14.0
[0.13.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.13.0
[0.12.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.12.0
[0.11.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.11.0
[0.10.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.10.0
[0.9.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.9.0
[0.8.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.8.0
[0.7.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.7.0
[0.6.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.6.0
[0.5.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.5.0
[0.4.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.4.0
[0.3.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.3.0
[0.2.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.2.0
