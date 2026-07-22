# Changelog

All notable changes to `github.com/iome-sh/iomesh-client-sdk-go` are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.66.0] — 2026-07-22

Minor release: pull-loop RESULT duration_ms/failed/strict always-emit.

### Changed

- **Example** `examples/pull-loop` — `RESULT=done` always emits `duration_ms=D` / `failed=true|false` / `strict=true|false` after `loops=` before `result=` (same wall-clock / hard-fail / IOMESH_STRICT knobs as SUMMARY; 0/false honest when zero/unset) so scrapers peer SUMMARY without inventing readiness; composes with existing identity / stream / consumer / batch / result / exit_code knobs

## [0.65.0] — 2026-07-22

Minor release: FormatKVEntry created_at always-emit.

### Changed

- **`FormatKVEntry`** — always emit `created_at` for operator/CI scrapers (RFC3339 UTC when set; blank value after the colon when zero/unset rather than omitting the line); peers FormatStreamDetail always-emit continuum; pure helper, no network I/O

## [0.64.0] — 2026-07-22

Minor release: FormatBucketInfo always-emit optional knobs.

### Changed

- **`FormatBucketInfo`** — always emit optional bucket knobs for operator/CI scrapers: `history`, `max_bytes`, `ttl_seconds` (`0` when history unset; `*int64` nil prints blank after the colon rather than omitting the line); peers FormatStreamDetail always-emit continuum; pure helper, no network I/O

## [0.63.0] — 2026-07-22

Minor release: FormatStreamDetail always-emit optional fields.

### Changed

- **`FormatStreamDetail`** — always emit optional stream knobs for operator/CI scrapers: `description`, `retention`, `partitions`, `max_msgs`, `max_age_sec`, `created_at`, `subjects` (empty string / `0` / blank value when unset; `*int64` nil prints blank after the colon rather than omitting the line; empty subjects list prints `subjects:` then `  (none)`); peers FormatConsumerInfo/Subscription `filter_subject` always-emit continuum; pure helper, no network I/O

## [0.62.0] — 2026-07-22

Minor release: FormatConsumerInfo/Subscription filter_subject always-emit.

### Changed

- **`FormatConsumerInfo` / `FormatSubscription`** — always emit `filter_subject:  %s` (empty string when unset / sparse 409 ConsumerInfo); peers TUI dogfood `consumer_filter` always-emit continuum; composes without inventing a filter when empty

## [0.61.0] — 2026-07-21

Minor release: pull-loop SUMMARY/RESULT batch, max_wait_ms, and loops.

### Added

- **Example** `examples/pull-loop` — `SUMMARY` and `RESULT=done` always emit `batch=N` / `max_wait_ms=M` / `loops=L` after `consumer=` (connect fetch knobs from `IOMESH_BATCH` / `IOMESH_MAX_WAIT_MS` / `IOMESH_LOOPS`; defaults 5 / 2000 / 1 after clamp; keys always present even if zero) before `cycles_completed=` (SUMMARY) / before `result=` (RESULT) so scrapers key batch/long-poll/loops without re-parsing the banner; composes with existing stream/consumer / identity / `result` / `exit_code` knobs (does not invent readiness)

## [0.60.0] — 2026-07-21

Minor release: pull-loop SUMMARY/RESULT stream and consumer.

### Added

- **Example** `examples/pull-loop` — `SUMMARY` and `RESULT=done` always emit `stream=S` / `consumer=C` after `workspace=` (connect config from `IOMESH_STREAM` / `IOMESH_CONSUMER`; defaults `EVENTS` / `sdk-pull-loop` from `env()`; empty string still emits `stream=` / `consumer=` if truly unset) before `cycles_completed=` (SUMMARY) / before `result=` (RESULT) so scrapers key the durable pull target without re-parsing the banner `stream=`/`consumer=` line; composes with existing identity / `result` / `exit_code` knobs

## [0.59.0] — 2026-07-21

Minor release: pull-loop SUMMARY/RESULT result=ok|err.

### Added

- **Example** `examples/pull-loop` — `SUMMARY` and `RESULT=done` always emit `result=ok|err` derived from the existing hard-fail flag `failed` (`result=ok` when `failed==false`; `result=err` when `failed==true`), peers ConnectionStatus.Result / AggregateConnectionResult; field order places `result=` after `strict=` (SUMMARY) / after `workspace=` (RESULT) before `exit_code=` so scrapers can key stage outcome without re-deriving from `failed` or exit status; composes with existing `failed` / `strict` / `exit_code` knobs (does not remove them)

## [0.58.0] — 2026-07-21

Minor release: pull-loop SUMMARY/RESULT base_url always-emit.

### Added

- **Example** `examples/pull-loop` — `SUMMARY` and `RESULT=done` always emit `base_url=B` after `user_agent=` before `tenant=` (connect mesh URL / `IOMESH_URL`, same string ConnectionStatus uses as `base_url`; empty string still emits `base_url=` if truly unset) so scrapers peer ConnectionStatus `base_url` without inventing readiness; field order matches on SUMMARY and RESULT for scrapers (`version=` `user_agent=` `base_url=` `tenant=` …)

## [0.57.0] — 2026-07-21

Minor release: pull-loop SUMMARY/RESULT user_agent always-emit.

### Added

- **Example** `examples/pull-loop` — `SUMMARY` and `RESULT=done` always emit `user_agent=UA` after `version=` before `tenant=` (package default `iomesh-client-sdk-go/<Version>`, same string ConnectionStatus uses when `WithUserAgent` is unset; empty string still emits `user_agent=` if truly unset) so scrapers peer ConnectionStatus `user_agent` without inventing readiness; field order matches on SUMMARY and RESULT for scrapers

## [0.56.0] — 2026-07-21

Minor release: pull-loop RESULT identity always-emit.

### Added

- **Example** `examples/pull-loop` — `RESULT=done` always emits `tenant=T` / `org=O` / `workspace=W` (same connect identity triple as SUMMARY from `IOMESH_TENANT` / `IOMESH_ORG` / `IOMESH_WORKSPACE`; empty string honest when unset) after `version=` so scrapers peer ConnectionStatus / SUMMARY identity on the RESULT line without inventing readiness; composes with existing `version` and `exit_code`

## [0.55.0] — 2026-07-21

Minor release: pull-loop SUMMARY identity always-emit.

### Added

- **Example** `examples/pull-loop` — `SUMMARY` always emits `tenant=T` / `org=O` / `workspace=W` (connect identity from `IOMESH_TENANT` / `IOMESH_ORG` / `IOMESH_WORKSPACE`; empty string honest when unset) after leading `version=` so scrapers peer ConnectionStatus identity without inventing readiness; composes with existing SUMMARY knobs

### Changed

- **`ConnectionStatus` probe error fields** — always-emitted JSON `health_err` / `ready_err` (empty string when probes OK / not set; no `omitempty`); `FormatConnectionStatus` always prints `health_err=` / `ready_err=` after `health=` / `ready=` (empty when OK; FAIL lines no longer inline `err=` to avoid duplication). Composes with existing identity / version / latencies / result always-emit continuum

## [0.54.0] — 2026-07-21

Minor release: ConnectionStatus identity always-emit.

### Changed

- **`ConnectionStatus` identity fields** — always-emitted JSON `tenant` / `org` / `workspace` (empty string when unset / nil client; no `omitempty`); `FormatConnectionStatus` always prints `tenant=` / `org=` / `workspace=` after `base_url` (composes with existing version / latencies / result)

## [0.53.0] — 2026-07-20

Minor release: pull-loop RESULT version.

### Added

- **Example** `examples/pull-loop` — `RESULT=done` always emits `version=V` (SDK package `Version` const, same as SUMMARY) so scrapers can key SDK version on the RESULT line without re-parsing SUMMARY or the banner `sdk=` line; composes with existing `exit_code=E`; empty version still emits `version=`

## [0.52.0] — 2026-07-20

Minor release: pull-loop SUMMARY version.

### Added

- **Example** `examples/pull-loop` — `SUMMARY` always emits leading `version=V` (SDK package `Version` const) so scrapers can key SDK version without re-parsing the banner `sdk=` line; composes with existing SUMMARY knobs

## [0.51.0] — 2026-07-20

Minor release: ConnectionStatus version evidence.

### Added

- **`ConnectionStatus.Version`** — always-emitted `version` (SDK package `Version` const, including nil client); `FormatConnectionStatus` prints `version=…` after `user_agent` (empty field falls back to package Version)

## [0.50.0] — 2026-07-20

Minor release: pull-loop RESULT exit_code.

### Added

- **Example** `examples/pull-loop` — `RESULT=done` always emits `exit_code=0|1` (same semantics as SUMMARY / process exit: `1` only when `strict && failed`; otherwise `0`, including non-strict with `failed=true`); keeps the `RESULT=done` token for scrapers that key off the RESULT line

## [0.49.0] — 2026-07-20

Minor release: pull-loop SUMMARY exit_code.

### Added

- **Example** `examples/pull-loop` — `SUMMARY` always emits `exit_code=0|1` (process exit after SUMMARY: `1` only when `strict && failed`, same condition as `os.Exit(1)`; otherwise `0`, including non-strict with `failed=true`)

## [0.48.0] — 2026-07-20

Minor release: pull-loop SUMMARY always emits wait_ready_attempts.

### Added

- **Example** `examples/pull-loop` — `SUMMARY` always emits `wait_ready_attempts=A` (probe cycle count from `WaitReadyAttempts` when WaitReady ran; `0` when WaitReady off, same zeroing as `wait_interval_ms`)

## [0.47.0] — 2026-07-20

Minor release: WaitReadyAttempts probe cycle count.

### Added

- **`WaitReadyAttempts`** — like `WaitReadyElapsed` but also returns probe attempt cycle count (each Ready [+ Health] try; incremented at start of each poll iteration); nil client → `(0, 0, error)`; success → `attempts >= 1`; cancel/timeout → attempts counted so far (`>= 0`); `WaitReadyElapsed` / `WaitReady` delegate to it
- **Example** `examples/pull-loop` — WaitReady preflight uses `WaitReadyAttempts`; PASS/WARN lines include `attempts=N`

## [0.46.0] — 2026-07-20

Minor release: pull-loop SUMMARY strict flag.

### Added

- **Example** `examples/pull-loop` — `SUMMARY` always emits `strict=true|false` (`IOMESH_STRICT` mode) so scrapers see whether hard-fail exit was enabled without re-parsing env/banner

## [0.45.0] — 2026-07-20

Minor release: pull-loop SUMMARY failed flag.

### Added

- **Example** `examples/pull-loop` — `SUMMARY` always emits hard-fail flag `failed=true|false` (same boolean used for `IOMESH_STRICT` exit) so scrapers see stage smoke failures even when non-strict (exit 0)

## [0.44.0] — 2026-07-20

Minor release: pull-loop SUMMARY WaitReady knobs.

### Added

- **Example** `examples/pull-loop` — `SUMMARY` always emits WaitReady knobs `wait_ready_ms=W wait_interval_ms=I wait_require_health=B` (budget / effective interval / require-health; `0` / `false` when WaitReady off so scrapers see knobs unused; when on, interval is the effective poll ms from `IOMESH_WAIT_INTERVAL_MS`)

## [0.43.0] — 2026-07-20

Minor release: pull-loop WaitReady interval env.

### Added

- **Example** `examples/pull-loop` — optional `IOMESH_WAIT_INTERVAL_MS` sets WaitReady poll interval (ms; default 500 when empty/invalid/≤0; clamp max 60000; only applied when `IOMESH_WAIT_READY_MS>0`); banner `wait_interval_ms=N`; PASS/WARN lines include `interval_ms=N`

## [0.42.0] — 2026-07-20

Minor release: pull-loop WaitReady require-health env.

### Added

- **Example** `examples/pull-loop` — optional `IOMESH_WAIT_REQUIRE_HEALTH=1` so WaitReady preflight (when `IOMESH_WAIT_READY_MS>0`) uses `WaitReadyOptions{RequireHealth: true}`; banner `wait_require_health=%v`; PASS/WARN lines include `require_health=%v`; default false


## [0.41.0] — 2026-07-20

Minor release: pull-loop STRICT uses ConnectionStatus.result.

### Changed

- **Example** `examples/pull-loop` — under `IOMESH_STRICT=1`, ConnectionStatus hard-fail uses aggregate `result=err` once (still prints per-probe PASS/WARN Health/Ready detail; `WARN ConnectionStatus result=err` when aggregate is err)


## [0.40.0] — 2026-07-20

Minor release: pull-loop WaitReady preflight.

### Added

- **Example** `examples/pull-loop` — optional `IOMESH_WAIT_READY_MS` WaitReady preflight after ConnectionStatus (budget ms; interval 500ms; `PASS WaitReady elapsed_ms=N` / `WARN WaitReady: … elapsed_ms=N`; banner `wait_ready_ms=N`; hard-fail under `IOMESH_STRICT=1` like Health)


## [0.39.0] — 2026-07-20

Minor release: ConnectionStatus aggregate result.

### Added

- **`ConnectionStatus.Result`** — always-emitted aggregate `result` (`ok` when both Health and Ready OK, otherwise `err`; includes nil client); `AggregateConnectionResult` pure helper; `FormatConnectionStatus` prints `result=ok|err` after `duration_ms`


## [0.38.0] — 2026-07-20

Minor release: ConnectionStatus duration_ms wall-clock.

### Added

- **`ConnectionStatus.DurationMS`** — always-emitted `duration_ms` wall clock for the full Health+Ready probe path (ms; `0` for nil client / not run); `FormatConnectionStatus` prints `duration_ms=N` after probe latencies


## [0.37.0] — 2026-07-20

Minor release: WaitReadyElapsed helper.

### Added

- **`WaitReadyElapsed`** — like `WaitReady` but also returns wall-clock wait duration until success or failure (nil client → `(0, error)`; elapsed always >= 0); `WaitReady` delegates to it


## [0.36.0] — 2026-07-20

Minor release: pull-loop IOMESH_STRICT exit mode.

### Added

- **Example** `examples/pull-loop` — optional `IOMESH_STRICT=1` exits non-zero (1) after `SUMMARY` when stage smoke hard failures occur (Health/Ready not OK, EnsureStream, PullSubscribe, Publish when requested, FetchContext, DeleteConsumer when requested); default remains warn-only + exit 0


## [0.35.0] — 2026-07-20

Minor release: ConnectionStatus health/ready probe latencies.

### Added

- **`ConnectionStatus` probe latencies** — always-emitted `health_ms` / `ready_ms` (Health and Ready wall time in ms; `0` for nil client / not run); `FormatConnectionStatus` prints `health_ms=N` / `ready_ms=N` for operator/CI evidence


## [0.34.0] — 2026-07-20

Minor release: FormatSubscription operator helper.

### Added

- **`FormatSubscription`** — pure operator helper for a pull subscription handle (nil → `"iomesh subscription: nil\n"`; otherwise stream/consumer from the handle plus FormatConsumerInfo body fields)
- **Example** `examples/pull-loop` — prints `FormatSubscription` after PullSubscribe


## [0.33.0] — 2026-07-20

Minor release: Subscription.Delete context helper.

### Added

- **`Subscription.Delete`** — context-aware wrapper that removes the durable consumer via `DeleteConsumer` (stream/name from the subscription; nil subscription / nil client → error)
- **Example** `examples/pull-loop` — `IOMESH_DELETE_CONSUMER=1` now uses `sub.Delete` after fetch loops


## [0.32.0] — 2026-07-20

Minor release: pull-loop publish-each cycle.

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
