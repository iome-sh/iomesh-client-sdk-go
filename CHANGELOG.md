# Changelog

All notable changes to `github.com/iome-sh/iomesh-client-sdk-go` are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Example** `examples/pull-loop` — optional `IOMESH_PUBLISH_EACH=1` publishes one message at the start of each fetch cycle (self-contained multi-fetch smoke); when set, pre-loop `IOMESH_PUBLISH` is skipped so the first cycle is not double-published; status banner includes `publish_each=`

## [0.31.0] — 2026-07-20

Minor release: DeleteConsumer and pull-loop cleanup.

### Added

- **`DeleteConsumer`** — `DELETE /v1/streams/{stream}/consumers/{name}` (204 success; path-escaped segments; empty stream/name / nil client → error; non-2xx → `*APIError`)
- **Example** `examples/pull-loop` — optional `IOMESH_DELETE_CONSUMER=1` best-effort cleanup after fetch loops (`PASS DeleteConsumer` / warn-only on fail)


## [0.30.0] — 2026-07-20

Minor release: pull-loop SUMMARY duration and cycle stats.

### Added

- **Example** `examples/pull-loop` — wall-clock `SUMMARY cycles_completed=N fetch_total=M duration_ms=D` before `RESULT=done` for stage smoke evidence (always printed)


## [0.29.0] — 2026-07-20

Minor release: pull-loop multi-fetch loops.

### Added

- **Example** `examples/pull-loop` — optional `IOMESH_LOOPS` multi-fetch cycles (default 1, max 100); each cycle `FetchContext` → `FormatMsgs` → optional `AckContext`; fetch error is warn-and-break with `RESULT=done`


## [0.28.0] — 2026-07-20

Minor release: pull-loop ensure-aware default consumer filter.

### Fixed

- **Example** `examples/pull-loop` — when `IOMESH_ENSURE_STREAM=1` and `IOMESH_SUBJECT` is unset, the durable pull consumer filter defaults to `stream.>` (matching EnsureStream subjects) so pull is scoped consistently with the ensure-aware default publish subject `stream.sdk-pull-loop`. Explicit `IOMESH_SUBJECT` still wins.

## [0.27.0] — 2026-07-19

Minor release: pull-loop ensure-aware default publish subject.

### Fixed

- **Example** `examples/pull-loop` — when `IOMESH_ENSURE_STREAM=1`, default publish subject is `stream.sdk-pull-loop` (under EnsureStream subjects `stream.>`) so `IOMESH_PUBLISH=1` works without setting `IOMESH_PUB_SUBJECT`. Explicit `IOMESH_PUB_SUBJECT` / `IOMESH_SUBJECT` still win.

## [0.26.0] — 2026-07-19

Minor release: pull-loop optional publish and CI example builds.

### Added

- **Example** `examples/pull-loop` — optional `IOMESH_PUBLISH=1` self-contained Publish before fetch (`IOMESH_PUB_SUBJECT` or derived subject; warn-only on publish fail)
- **CI** — build all `./examples/...` after tests so examples stay compile-clean

## [0.25.0] — 2026-07-19

Minor release: examples/pull-loop pull consumer demo.

### Added

- **Example** `examples/pull-loop` — stage smoke for durable pull consumer (`ConnectionStatus`, optional `EnsureStream`, `PullSubscribe` / `FetchContext` / `FormatMsgs` / optional `AckContext`)

## [0.24.0] — 2026-07-19

Minor release: FormatMsgs and pull-loop docs.

### Added

- **`FormatMsgs`** — pure operator helper for a batch of fetched messages (`count` header + one `FormatMsg` line per message; nil/empty → `count=0`)
- **Docs** — godoc `ExampleFormatMsgs` / pull-loop example; README pull-loop snippet (`FetchContext` + `FormatMsgs` + `AckContext`)

## [0.23.0] — 2026-07-19

Minor release: Subscription FetchContext, AckContext, NackContext.

### Added

- **`Subscription.FetchContext` / `AckContext` / `NackContext`** — context-aware pull subscription ops so callers need not use `context.Background()`. Existing `Fetch` / `Ack` / `Nack` remain as thin wrappers that pass `context.Background()`
- **`DefaultFetchMaxWait`** — exported `5s` default long-poll wait for `Fetch` / `FetchContext` / `ConsumerFetch` when `MaxWait` is omitted
- **`FormatMsg`** — pure operator helper for one fetched message (`seq`, `subject`, byte length)

### Changed

- **`MaxWait` / `Fetch` docs** — document default via `DefaultFetchMaxWait`

## [0.22.0] — 2026-07-19

Minor release: ConsumerFetch, ConsumerAck, and ConsumerNack client helpers.

### Added

- **`ConsumerFetch` / `ConsumerAck` / `ConsumerNack`** — standalone one-shot consumer ops (`POST …/fetch|ack|nack`) without holding a long-lived `Subscription`. Path-escape stream and consumer segments. `ConsumerFetch` wires returned `Msg` values to an ephemeral subscription so `Msg.Ack` / `Msg.Nack` work

### Changed

- **`Subscription.Fetch` / `Ack` / `Nack`** — thin wrappers over the client helpers (still use `context.Background()`)

## [0.21.0] — 2026-07-19

Minor release: CreateConsumer and EnsureConsumer helpers.

### Added

- **`CreateConsumer` / `EnsureConsumer`** — standalone durable pull consumer helpers (`POST /v1/streams/{stream}/consumers`); return `*ConsumerInfo`. On 201, full body decode; on 409 conflict, success with `&ConsumerInfo{Stream, Name}` (name-only). `EnsureConsumer` is an idempotent alias of `CreateConsumer`
- **`CreateConsumerConfig`** — public config type (`Stream`, `Name`, `FilterSubject`, `MaxDeliver`, `AckWaitSec`)

### Changed

- **`PullSubscribe`** — refactored to use `CreateConsumer`; 409 reuse now carries Stream/Name (not fully zero info)

## [0.20.0] — 2026-07-19

Minor release: PullSubscribe ConsumerInfo and path escape.

### Added

- **`ConsumerInfo`** — public durable-consumer metadata type (`stream`, `name`, `ack_floor`, `pending_count`, `filter_subject`)
- **`Subscription.ConsumerInfo()`** — returns create-response metadata (zero value after 409 reuse)
- **`FormatConsumerInfo`** — pure operator helper for consumer metadata detail (no network I/O)

### Changed

- **`PullSubscribe`** — on 201, decodes `ConsumerInfo` from the create response into the subscription; on 409 conflict, success with empty/zero info. Stream path segment is `url.PathEscape`'d
- **`Subscription.Fetch` / `Ack` / `Nack`** — path-escape stream and consumer URL segments

## [0.19.0] — 2026-07-19

Minor release: EnsureBucket and FormatBucketInfo.

### Added

- **`EnsureBucket`** — documented idempotent alias of `CreateBucket` (same signature; 409 conflict is success)
- **`FormatBucketInfo`** — pure operator helper for bucket metadata detail (no network I/O)

## [0.18.0] — 2026-07-19

Minor release: Put returns PutResult with revision metadata. Pre-1.0 signature change.

### Added

- **`PutResult`** — public put outcome type (`bucket`, `key`, `revision`) returned by `Put`
- **`FormatPutResult`** — pure operator helper for put outcome detail (no network I/O)

### Changed

- **`Put`** — now returns `(*PutResult, error)` instead of `(uint64, error)`. Decodes full put metadata from the response body; defensive fill of bucket/key from args when the broker omits them. Empty bucket/key / nil client → error. **Breaking for pre-1.0 callers** that assigned a revision `uint64` — update to two-value `*PutResult` assign

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

[Unreleased]: https://github.com/iome-sh/iomesh-client-sdk-go/compare/v0.26.0...HEAD
[0.26.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.26.0
[0.25.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.25.0
[0.24.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.24.0
[0.23.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.23.0
[0.22.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.22.0
[0.21.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.21.0
[0.20.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.20.0
[0.19.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.19.0
[0.18.0]: https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.18.0
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
