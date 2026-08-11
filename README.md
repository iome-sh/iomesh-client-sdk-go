# I/O Mesh Client SDK for Go

[![CI](https://github.com/iome-sh/iomesh-client-sdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/iome-sh/iomesh-client-sdk-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/iome-sh/iomesh-client-sdk-go.svg)](https://pkg.go.dev/github.com/iome-sh/iomesh-client-sdk-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Official **Go client SDK** for the [I/O Mesh](https://iome.sh) broker and connector platform.

Publish and pull **organizational heartbeats** (ops **pulse**) on `dept.*` streams: connectors and services emit org-tool events; agents and workers consume them via durable pull. Public lexicon is **heartbeat / pulse** only.

Memory helpers stay **local-primary** honest: durable stream paths first; `DualWriteMemoryTurn` defaults to **async-only** (`Sync: false` / dual_write **OFF**) — optional sidecar audit, not a freemium hosted palace. Offline stage smoke ≠ live APPLY. Surfaces are **Beta** / pre-1.0 — do not invent GA.

Official open-source tooling from [IOMesh](https://iome.sh) (**IOMesh Technology Ltd.**). This repository is **MIT edge client code** only — not free mesh control-plane access.

| Capability | Package |
|------------|---------|
| HTTP publish / pull subscribe / streams / KV / memory (org heartbeats on `dept.*`) | [`iomeshclient`](./iomeshclient) |
| Partner webhook HMAC + observation envelopes | [`connectorsdk`](./connectorsdk) |
| Kafka protocol (Produce subset) | [`kafka`](./kafka) · via `iomeshclient.KafkaClient` |
| Shared envelope + CUID helpers | [`envelope`](./envelope) · [`cuid`](./cuid) |

> **Module path:** `github.com/iome-sh/iomesh-client-sdk-go`  
> **Package:** `iomeshclient`  
> **Env prefix:** `IOMESH_*`  
> **Wire headers:** `X-IOMesh-Tenant`, `X-IOMesh-Org`, `X-IOMesh-Workspace`, …  
> **Status:** public OSS **v0.67.x** (pre-1.0, **Beta**). Memory M2/M3 + multi-tenant headers + dual-write/metering + Health/Ready/WaitReady + catalog plane + EvaluatePolicy + QueryContext + ConnectionStatus + ListStreams/GetStream/DeleteStream/ListStreamMessages + CreateStream/EnsureStream `*StreamInfo` + FormatStreams/FormatStreamDetail + CreateConsumer/EnsureConsumer `*ConsumerInfo` + ConsumerFetch/ConsumerAck/ConsumerNack + PullSubscribe `FetchContext`/`AckContext`/`NackContext` + `DefaultFetchMaxWait` + FormatMsg/FormatMsgs/FormatConsumerInfo + KV CreateBucket/EnsureBucket `*BucketInfo` + Put `*PutResult` + FormatBucketInfo/FormatKVEntry/FormatKVKeys/FormatPutResult aligned with [iomesh-tui](https://github.com/iome-sh/iomesh-tui). Always-emit format helpers are operator diagnostics — not new product APIs / not invent GA.  
> **User-Agent:** `iomesh-client-sdk-go/<Version>` (override with `WithUserAgent`).

## Requirements

- Go **1.22+** (module declares the toolchain used in CI)
- Network access to an I/O Mesh broker (or local foundation)

## Install

```bash
go get github.com/iome-sh/iomesh-client-sdk-go@latest
```

## Quick start — publish an org heartbeat

Connect, ensure a stream under `dept.*`, and publish a single organizational heartbeat (ops pulse). Agents and workers pull the same subjects as durable consumers.

```go
package main

import (
	"context"
	"log"

	"github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
)

func main() {
	nc, err := iomeshclient.Connect(
		iomeshclient.Options{URL: "http://127.0.0.1:8422"},
		iomeshclient.WithTenant("dept.engineering"),
		iomeshclient.WithOrg("acme-org"),
		iomeshclient.WithWorkspace("ws_default"), // multi-tenant metering / entitlements
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	info, err := nc.CreateStream(ctx, iomeshclient.StreamConfig{
		Name:     "EVENTS",
		Subjects: []string{"dept.engineering.events.>"},
	})
	if err != nil {
		log.Fatal(err)
	}
	if info != nil {
		log.Printf("stream=%s subjects=%v", info.Name, info.Subjects)
	}

	// Organizational heartbeat (ops pulse) — public lexicon: heartbeat / pulse only.
	ack, err := nc.Publish(ctx, "EVENTS", "dept.engineering.events.demo", []byte(`{"hello":"mesh","kind":"org_heartbeat"}`))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("published seq=%d subject=%s partition=%d", ack.Seq, ack.Subject, ack.Partition)
}
```

Runnable framing (publish + optional pull): [`examples/org-heartbeat-publish/`](examples/org-heartbeat-publish/).  
Stage smoke pull loop: [`examples/pull-loop/`](examples/pull-loop/) (same durable consumer APIs; offline smoke ≠ live APPLY).

## Connector SDK (HMAC + envelope)

```go
import "github.com/iome-sh/iomesh-client-sdk-go/connectorsdk"

payload, err := connectorsdk.NormalizeEnvelope(
	"acme-crm", "engineering", "acme-crm", "evt-42", "contact.created",
	json.RawMessage(`{"email":"user@example.com"}`),
)
```

See [`examples/connector-sdk-template/`](examples/connector-sdk-template/) for a full webhook adapter (`IOMESH_URL`, `IOMESH_ORG`, …).

## Kafka Produce

```go
kc := iomeshclient.NewKafkaClient("127.0.0.1:9423")
defer kc.Close()

offset, err := kc.Produce(ctx, "mesh.finance.events", 0, []byte("key"), []byte(`{"event_id":"evt-1"}`))
```

## Streams

| API | Path | Notes |
|-----|------|--------|
| `CreateStream` / `EnsureStream` | `POST /v1/streams` | Returns `*StreamInfo`; 409 conflict → success + best-effort GET (nil info OK) |
| `ListStreams` | `GET /v1/streams` | Explicit discovery; non-2xx → `*APIError` (not fail-open empty) |
| `GetStream` | `GET /v1/streams/{name}` | Single `StreamInfo`; 404 → `*APIError` |
| `DeleteStream` | `DELETE /v1/streams/{name}` | 204 success; 404 → `*APIError`; destructive — not used in dogfood by default |
| `ListStreamMessages` | `GET /v1/streams/{name}/messages` | Stream replay/read-range; `from_seq`/`to_seq`/`limit`; payload base64→`[]byte`; non-2xx → `*APIError` |
| `CreateConsumer` / `EnsureConsumer` | `POST /v1/streams/{stream}/consumers` | Returns `*ConsumerInfo`; 409 conflict → success with Stream/Name only. EnsureConsumer is an idempotent alias |
| `DeleteConsumer` | `DELETE /v1/streams/{stream}/consumers/{name}` | 204 success; 404 → `*APIError`; destructive — opt-in cleanup (e.g. pull-loop `IOMESH_DELETE_CONSUMER=1`) |
| `ConsumerFetch` / `ConsumerAck` / `ConsumerNack` | `POST …/fetch\|ack\|nack` | One-shot ops without holding a `Subscription`; path-escape stream/consumer; Fetch wires `Msg.Ack`/`Msg.Nack` via ephemeral sub |
| `Publish` / `PullSubscribe` | stream publish / consumer | `PullSubscribe` uses `CreateConsumer` then returns `*Subscription` with `ConsumerInfo()`; `FetchContext`/`AckContext`/`NackContext` (or `Fetch`/`Ack`/`Nack` → `context.Background()`); `Delete(ctx)` removes the durable consumer via `DeleteConsumer`; default long-poll `DefaultFetchMaxWait` (5s) / `MaxWait`; path segments escaped |
| `FormatMsg` / `FormatMsgs` / `FormatConsumerInfo` / `FormatSubscription` / `FormatStreamDetail` | — | Pure operator helpers for one message / batch / consumer detail / subscription handle / stream detail (no network I/O); `FormatConsumerInfo` / `FormatSubscription` always emit `filter_subject` (empty when unset); `FormatStreamDetail` always emits description/retention/partitions/max_msgs/max_age_sec/created_at/subjects (blank/0/`(none)` when unset) |
| `Pub` | `POST /v1/pub` | Ephemeral fire-and-forget |

```go
// List all streams (callers handle errors — not fail-open)
streams, err := nc.ListStreams(ctx)
if err != nil {
	log.Fatal(err) // *iomeshclient.APIError on non-2xx
}
// streams[i].Name, Subjects, Messages, FirstSeq, LastSeq, CreatedAt, …
fmt.Print(iomeshclient.FormatStreams(streams)) // compact operator table

info, err := nc.GetStream(ctx, "EVENTS")
if err != nil {
	log.Fatal(err)
}
fmt.Print(iomeshclient.FormatStreamDetail(*info)) // multi-line detail (always-emits optional knobs)

// DeleteStream is destructive — opt-in only (e.g. IOMESH_DELETE_STREAM=name); not auto-run in dogfood
if err := nc.DeleteStream(ctx, "TEMP_STREAM"); err != nil {
	log.Fatal(err) // *iomeshclient.APIError on 404 / non-2xx
}

// Replay/read-range (defaults: from_seq=1, to_seq=0 last, limit=100 max 1000)
msgs, err := nc.ListStreamMessages(ctx, "EVENTS", iomeshclient.ListStreamMessagesOptions{
	FromSeq: 1,
	Limit:   50,
})
if err != nil {
	log.Fatal(err)
}
// msgs[i].Seq, Subject, Payload ([]byte), Headers, Timestamp, …

// CreateConsumer / EnsureConsumer: durable pull consumer (201 → full info; 409 → Stream/Name only)
info, err := nc.EnsureConsumer(ctx, iomeshclient.CreateConsumerConfig{
	Stream: "EVENTS", Name: "worker-1", FilterSubject: "dept.events.>",
})
if err != nil {
	log.Fatal(err)
}
fmt.Print(iomeshclient.FormatConsumerInfo(*info)) // operator detail

// DeleteConsumer is destructive — opt-in only (e.g. IOMESH_DELETE_CONSUMER=1 in pull-loop)
if err := nc.DeleteConsumer(ctx, "EVENTS", "worker-1"); err != nil {
	log.Fatal(err) // *iomeshclient.APIError on 404 / non-2xx
}

// PullSubscribe: CreateConsumer + subscription handle for Fetch/Ack/Nack/Delete
sub, err := nc.PullSubscribe(ctx, iomeshclient.PullSubscribeConfig{
	Stream: "EVENTS", Consumer: "worker-1", Filter: "dept.events.>",
})
if err != nil {
	log.Fatal(err)
}
fmt.Print(iomeshclient.FormatSubscription(sub)) // handle: stream/consumer + consumer info fields
// Or: fmt.Print(iomeshclient.FormatConsumerInfo(sub.ConsumerInfo()))

// Prefer FetchContext when you already have a request-scoped ctx (cancellation/deadlines).
// MaxWait defaults to DefaultFetchMaxWait (5s); override with MaxWait(d).
batch, err := sub.FetchContext(ctx, 10, iomeshclient.MaxWait(2*time.Second))
if err != nil {
	log.Fatal(err)
}
// batch[i].Ack() / Nack(), or: sub.AckContext(ctx, seqs...); sub.NackContext(ctx, seqs...)
// Fetch/Ack/Nack remain as Background wrappers for simple call sites.
fmt.Print(iomeshclient.FormatMsg(batch[0]))  // one message: seq / subject / bytes
fmt.Print(iomeshclient.FormatMsgs(batch))    // batch: count header + one line per msg

// Delete removes the durable consumer (same as DeleteConsumer with stream/name from sub).
// Destructive — opt-in only (e.g. IOMESH_DELETE_CONSUMER=1 in pull-loop).
if err := sub.Delete(ctx); err != nil {
	log.Fatal(err)
}

// Pull loop (FetchContext → FormatMsgs → AckContext)
// for {
//     batch, err := sub.FetchContext(ctx, 10)
//     if err != nil { log.Fatal(err) }
//     if len(batch) == 0 { continue }
//     fmt.Print(iomeshclient.FormatMsgs(batch))
//     seqs := make([]uint64, len(batch))
//     for i, m := range batch { seqs[i] = m.Seq() }
//     if err := sub.AckContext(ctx, seqs...); err != nil { log.Fatal(err) }
// }
// Runnable stage smoke: examples/pull-loop (IOMESH_URL, optional IOMESH_ENSURE_STREAM / IOMESH_PUBLISH / IOMESH_PUBLISH_EACH / IOMESH_LOOPS / IOMESH_ACK / IOMESH_DELETE_CONSUMER / IOMESH_WAIT_READY_MS / IOMESH_WAIT_INTERVAL_MS / IOMESH_WAIT_REQUIRE_HEALTH / IOMESH_STRICT;
// with ENSURE_STREAM=1, default filter is stream.> and pub subject is stream.sdk-pull-loop)

// One-shot consumer ops (no long-lived Subscription)
msgs, err := nc.ConsumerFetch(ctx, "EVENTS", "worker-1", 10)
if err != nil {
	log.Fatal(err)
}
// msgs[i].Ack() / Nack() work via ephemeral sub wiring
// or: nc.ConsumerAck(ctx, "EVENTS", "worker-1", seqs...); nc.ConsumerNack(...)
```

## KV (buckets + keys)

| API | Path | Notes |
|-----|------|--------|
| `CreateBucket` / `EnsureBucket` | `POST /v1/kv/{name}` | Returns `*BucketInfo`; 409 conflict → success with name only. EnsureBucket is an idempotent alias of CreateBucket |
| `Put` / `Get` / `Delete` | `/v1/kv/{bucket}/{key}` | Put returns `*PutResult` (revision metadata); value is base64 in JSON body; Get returns `*KVEntry` |
| `ListKeys` | `GET /v1/kv/{bucket}?prefix=` | Optional prefix filter |
| `FormatBucketInfo` / `FormatKVEntry` / `FormatKVKeys` / `FormatPutResult` | — | Pure operator format helpers (no network I/O); `FormatBucketInfo` always emits history/max_bytes/ttl_seconds (`0` / blank when unset) |

```go
info, err := nc.EnsureBucket(ctx, "agent-state", iomeshclient.CreateBucketConfig{
	History: 5,
})
if err != nil {
	log.Fatal(err)
}
if info != nil {
	fmt.Print(iomeshclient.FormatBucketInfo(*info)) // multi-line bucket detail (always-emits optional knobs)
}

put, err := nc.Put(ctx, "agent-state", "worker-1.checkpoint", []byte("seq=42"))
if err != nil {
	log.Fatal(err)
}
fmt.Print(iomeshclient.FormatPutResult(*put)) // bucket/key/revision

entry, err := nc.Get(ctx, "agent-state", "worker-1.checkpoint")
fmt.Print(iomeshclient.FormatKVEntry(*entry)) // multi-line entry detail

keys, err := nc.ListKeys(ctx, "agent-state", "worker-")
fmt.Print(iomeshclient.FormatKVKeys("agent-state", keys)) // compact key listing
```

## Memory (async streams + optional sync sidecar)

**Honesty:** local-primary · `DualWriteMemoryTurn` defaults to **async-only** (`Sync: false` = dual_write **OFF**) · optional sidecar audit · not primary freemium hosted palace · local AI ≠ platform GPU product · offline stage smoke ≠ live APPLY · Beta surfaces — no invent GA.

| API | Path | Notes |
|-----|------|--------|
| `PublishMemoryIngest` | `MEMORY_INGEST` publish | Async durable stream; temporal fields on `MemoryEnvelope` |
| `DualWriteMemoryTurn` | async + optional sync | Stream first; **default OFF** (no sync). Optional fail-open `IngestMemoryTurn` when `Sync: true` |
| `RequestMemoryRecall` / `RequestMemoryRecallFull` | `MEMORY_RPC` publish | Async; Full adds `session_id` correlation |
| `RetrieveMemory` | `POST /v1` then `/v5/memory/retrieve` | Sync hits; empty query OK if `session_id` set |
| `IngestMemoryTurn` | `POST /v1` then `/v5/memory/ingest` | Optional sync turn write (not freemium palace default) |

```go
// Sync retrieve (sidecar URL or gateway that routes /v1|/v5/memory/*)
hits, err := nc.RetrieveMemory(ctx, iomeshclient.MemoryRetrieveRequest{
	TenantID:  "dept.research",
	Query:     "lease rotation",
	SessionID: "dept.research.mesh-dogfood",
	Limit:     8,
})
// hits.Path is "/v1/memory/retrieve" or "/v5/memory/retrieve"

// Dual-write: durable stream first; Sync defaults OFF (local-primary / dual_write OFF).
// Optional Sync: true = best-effort sidecar audit (fail-open) — not primary freemium palace.
mesh, _ := iomeshclient.Connect(iomeshclient.Options{URL: os.Getenv("IOMESH_URL")}, /* tenant/org… */)
// Default path — async only (recommended):
res, err := mesh.DualWriteMemoryTurn(ctx, "dept.research", iomeshclient.MemoryEnvelope{
	Role: "user", Content: "decision notes", SessionID: "sess-1", SessionSeq: 1,
}, iomeshclient.DualWriteMemoryOptions{}) // Sync: false
// Optional audit path:
// palace, _ := iomeshclient.Connect(iomeshclient.Options{URL: os.Getenv("IOMESH_MEMORY_ENDPOINT")})
// res, err = mesh.DualWriteMemoryTurn(ctx, "dept.research", env, iomeshclient.DualWriteMemoryOptions{Sync: true, SyncClient: palace})
// res.Async is PubAck; res.SyncErr is set when optional sync fail-opens
```

The agent harness ([iomesh-tui](https://github.com/iome-sh/iomesh-tui)) mirrors these surfaces without depending on this module (lean public HTTP).

## Edge Memory OSS vs mesh memory HTTP

Three planes — residual-honest (s1479). This SDK is the **mesh/platform HTTP** client only. It does **not** import `github.com/iome-sh/memory` (palace kernel) and does **not** re-implement a local FS palace or attach MCP stdio.

| Plane | Package / surface | Role |
|-------|-------------------|------|
| Local edge palace | [`iomesh-memory-mcp`](https://github.com/iome-sh/iomesh-memory-mcp) + [`github.com/iome-sh/memory`](https://github.com/iome-sh/memory) | Customer-local MCP host + palace kernel · **dual_write OFF** · **not Memory GA** · local-primary FS palace |
| Mesh / platform HTTP | this SDK — `RetrieveMemory` / `IngestMemoryTurn` / streams | Broker/gateway (and optional memory **sidecar HTTP**) paths · **not** local FS palace |
| Private broker | aion monorepo | Streaming mesh + CP + residual private hosts · **stays private** |

Public install (no `GOPRIVATE` required for the edge modules):

```bash
go install github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main
go get github.com/iome-sh/memory@main
```

**Honesty locks**

- **dual_write OFF** by default on SDK helpers (`DualWriteMemoryTurn` → async-only unless `Sync: true`).
- **not Memory GA** — edge OSS + SDK memory helpers are Beta / residual; do not invent product GA.
- **local-primary ≠ freemium palace** — customer-local MCP is not a hosted freemium Memory product.
- **SDK HTTP ≠ invent local MCP attach** — `IOMESH_URL` / `IOMESH_MEMORY_ENDPOINT` target mesh or sidecar **HTTP**; they do not open MCP stdio or bind a local palace process for you.
- **aion stays private** — do not treat monorepo broker/CP as a public dependency of this SDK.
- Cross-links: [iomesh-tui](https://github.com/iome-sh/iomesh-tui) (agent edge) · [iomesh-memory-mcp](https://github.com/iome-sh/iomesh-memory-mcp) (public edge host) · [memory](https://github.com/iome-sh/memory) (public kernel).

## Metering (dept streams / org-tool heartbeats)

`EmitDeptEvent` / `EmitLLMCall` publish structured **organizational tool heartbeats** on the `dept` stream (`dept.*` subjects). Agents and ops dashboards consume these pulses; they are not wearable/medical brand claims.

```go
// Remote multi-tenant usage event (org-tool heartbeat) for platform dashboards
ack, err := nc.EmitLLMCall(ctx, iomeshclient.LLMCallEvent{
	Tenant: "dept.research", SessionID: "sess-1",
	Model: "deepseek-v4-flash", TotalTokens: 120, EstUSD: 0.002,
})
// Wire: POST /v1/streams/dept/publish subject=dept.agent.llm_call
```

Stage smoke (mesh + optional memory sidecar; dual_write sync only when `IOMESH_MEMORY_ENDPOINT` differs):

```bash
export IOMESH_URL=http://127.0.0.1:8422
export IOMESH_MEMORY_ENDPOINT=http://127.0.0.1:8765  # warm plane
# optional: IOMESH_PREFER_SHORTER_HOPS=0|false for legacy related sort; omit/1|true = PreferShorterHops
# (omit = nil → kernel default true). Multi-hop lite · not full graph RAG · not Memory GA · dual_write OFF.
go run ./examples/memory-metering-dogfood
```

Pull consumer stage smoke (one or more fetch cycles; optional ensure/publish/ack):

```bash
export IOMESH_URL=http://127.0.0.1:8422
export IOMESH_STREAM=EVENTS
export IOMESH_CONSUMER=sdk-pull-loop
# export IOMESH_ENSURE_STREAM=1  # create stream with subject stream.>
# export IOMESH_PUBLISH=1        # publish one demo message before the fetch loop
# export IOMESH_PUBLISH_EACH=1   # publish one message at the start of each cycle
# export IOMESH_LOOPS=3          # multi-fetch cycles (default 1, max 100)
# export IOMESH_ACK=1            # ack fetched sequences each cycle
# export IOMESH_DELETE_CONSUMER=1  # best-effort sub.Delete after fetch loops
# export IOMESH_WAIT_READY_MS=5000  # optional WaitReady preflight budget (ms) after ConnectionStatus
# export IOMESH_WAIT_INTERVAL_MS=250  # optional WaitReady poll interval (ms; default 500; only when wait_ready_ms>0)
# export IOMESH_WAIT_REQUIRE_HEALTH=1  # optional; WaitReady also requires Health (only when wait_ready_ms>0)
# export IOMESH_STRICT=1         # exit 1 after SUMMARY on hard stage failures (probe aggregate via ConnectionStatus.result)
go run ./examples/pull-loop
# ends with:
# SUMMARY version=V user_agent=UA base_url=B tenant=T org=O workspace=W stream=S consumer=C cycles_completed=N fetch_total=M duration_ms=D wait_ready_ms=W wait_interval_ms=I wait_require_health=B wait_ready_attempts=A failed=F strict=S result=R exit_code=E
# RESULT=done version=V user_agent=UA base_url=B tenant=T org=O workspace=W stream=S consumer=C result=R exit_code=E
```

With `IOMESH_ENSURE_STREAM=1`, the consumer filter defaults to `stream.>` (matching EnsureStream subjects) and with `IOMESH_PUBLISH=1` / `IOMESH_PUBLISH_EACH=1` the default publish subject is `stream.sdk-pull-loop` so Publish is accepted without setting `IOMESH_PUB_SUBJECT`. Override filter/pub with `IOMESH_SUBJECT` / `IOMESH_PUB_SUBJECT`. `IOMESH_PUBLISH=1` alone publishes once before the loop; `IOMESH_PUBLISH_EACH=1` publishes at the start of each cycle (and skips the pre-loop publish when both are set, so the first cycle is not double-published). Set `IOMESH_DELETE_CONSUMER=1` for best-effort `sub.Delete` after fetch loops (`PASS` / warn-only). Set `IOMESH_WAIT_READY_MS=N` (N>0) for an optional `WaitReadyAttempts` preflight after ConnectionStatus (budget N ms; poll interval from `IOMESH_WAIT_INTERVAL_MS`, default 500ms when empty/invalid/≤0, clamp max 60000; prints `PASS WaitReady elapsed_ms=… interval_ms=… require_health=… attempts=…` or `WARN WaitReady: … elapsed_ms=… interval_ms=… require_health=… attempts=…`; banner shows `wait_ready_ms=N`, `wait_interval_ms=N`, and `wait_require_health=%v`, `wait_ready_ms=0` when off). Set `IOMESH_WAIT_REQUIRE_HEALTH=1` so that preflight uses `WaitReadyOptions{RequireHealth: true}` (only applies when `IOMESH_WAIT_READY_MS>0`; default false). Always prints `SUMMARY` (leading `version=V` from package `Version` + always-emitted `user_agent=UA` package default `iomesh-client-sdk-go/<Version>` after `version=` before `base_url=` (same string ConnectionStatus uses when `WithUserAgent` is unset; empty string still emits `user_agent=` if truly unset) + always-emitted `base_url=B` from connect mesh URL / `IOMESH_URL` after `user_agent=` before `tenant=` (same string ConnectionStatus uses as `base_url`; empty string still emits `base_url=` if truly unset) + always-emitted connect identity `tenant=T` / `org=O` / `workspace=W` from `IOMESH_TENANT` / `IOMESH_ORG` / `IOMESH_WORKSPACE` (empty string honest when unset) + always-emitted `stream=S` / `consumer=C` from `IOMESH_STREAM` / `IOMESH_CONSUMER` (defaults `EVENTS` / `sdk-pull-loop`; empty string honest if truly unset) after `workspace=` before `cycles_completed=` + cycle/fetch counts + wall-clock `duration_ms` + WaitReady knobs `wait_ready_ms` / `wait_interval_ms` / `wait_require_health` / `wait_ready_attempts` + hard-fail flag `failed=true|false` + `strict=true|false` for `IOMESH_STRICT` mode + always-emitted `result=ok|err` derived from `failed` (`ok` when `failed==false`; `err` when `failed==true`; peers ConnectionStatus.Result) + `exit_code=0|1` matching process exit after SUMMARY; when WaitReady is off the knobs are `0` / `0` / `false` / `0`) then `RESULT=done version=V user_agent=UA base_url=B tenant=T org=O workspace=W stream=S consumer=C result=R exit_code=E` (same `version`, `user_agent`, `base_url`, identity, `stream`, `consumer`, `result`, and `exit_code` semantics as SUMMARY for scrapers that key off the RESULT line; empty identity/base_url/stream/consumer strings honest when unset). Set `IOMESH_STRICT=1` so hard stage failures (`ConnectionStatus.result=err` for Health/Ready probe aggregate, WaitReady when requested, EnsureStream, PullSubscribe, Publish when requested, FetchContext, DeleteConsumer when requested) exit non-zero (1) after `SUMMARY` / `RESULT`; default remains warn-only + exit 0 (`failed` still reflects hard stage failures for scrapers; `result` mirrors `failed` as `ok|err`; `strict` reflects whether hard-fail exit mode was enabled; `exit_code=1` only when `strict && failed`, otherwise `0`).

See [`examples/pull-loop/`](examples/pull-loop/) for env flags (`IOMESH_BATCH`, `IOMESH_MAX_WAIT_MS`, `IOMESH_LOOPS`, `IOMESH_SUBJECT`, `IOMESH_PUBLISH`, `IOMESH_PUBLISH_EACH`, `IOMESH_DELETE_CONSUMER`, `IOMESH_WAIT_READY_MS`, `IOMESH_WAIT_INTERVAL_MS`, `IOMESH_WAIT_REQUIRE_HEALTH`, `IOMESH_STRICT`, …).

## Diagnostics

```go
fmt.Println(iomeshclient.Version) // e.g. "0.26.0"
// All requests send: User-Agent: iomesh-client-sdk-go/0.26.0
// Override: iomeshclient.WithUserAgent("my-service/1.2.3")

if err := nc.Health(ctx); err != nil { /* broker down */ }
if err := nc.Ready(ctx); err != nil { /* optional readiness path missing */ }

// One-shot identity + Health + Ready (fail-open; never panics). Both probes always run.
// Always includes tenant / org / workspace (empty string when unset / nil client).
// Always includes version (package Version const, including nil client).
// Always includes health_err / ready_err (empty string when probes OK).
// Always includes health_ms / ready_ms / duration_ms (probe wall time ms; 0 when nil client / not run).
// duration_ms is wall clock for the full Health+Ready path.
// result is always "ok" | "err" (both probes OK → ok; otherwise err, including nil client).
st := nc.ConnectionStatus(ctx)
fmt.Print(iomeshclient.FormatConnectionStatus(st))
// or: fmt.Print(iomeshclient.FormatConnectionStatusJSON(st))


// Poll until Ready (optional Health) or ctx deadline.
if err := nc.WaitReady(ctx, iomeshclient.WaitReadyOptions{
	Interval: 500 * time.Millisecond, // default when zero
	// RequireHealth: true,
}); err != nil { /* still not ready */ }
// Or capture wait latency (wall time until success or error):
// elapsed, err := nc.WaitReadyElapsed(ctx, iomeshclient.WaitReadyOptions{Interval: 500 * time.Millisecond})
// Or also capture probe attempt cycles (each Ready [+ Health] try):
// elapsed, attempts, err := nc.WaitReadyAttempts(ctx, iomeshclient.WaitReadyOptions{Interval: 500 * time.Millisecond})

// Optional remote policy (POST /v1/policy/evaluate). Mode is per-call.
// Transport / 404 / non-OK are fail-open (Allow=true) so agent DX is not blocked
// when the broker is down or the endpoint is not deployed yet.
// Enforce only blocks via ShouldBlockTool when mesh explicitly denies (Source=mesh).
dec := nc.EvaluatePolicy(ctx, iomeshclient.PolicyInput{
	Tool: "run_shell",
	Mode: iomeshclient.PolicyEnforce, // or PolicyAdvisory; empty/off skips network
})
if dec.ShouldBlockTool() {
	// mesh deny under enforce
}
_ = dec.Summary() // e.g. "allow source=mesh mode=enforce"

// Context plane (POST /v1/context/query). Fail-open: nil client / transport / non-OK → empty.
// ContextSnippet always requests include_lineage for agent prompt injection.
snip := nc.ContextSnippet(ctx, ".", "incidents last hour")
// or:
res := nc.QueryContext(ctx, iomeshclient.QueryContextRequest{
	Workspace: ".", Query: "incidents", IncludeLineage: true,
})
_ = iomeshclient.FormatContextSnippet(res) // text + optional <iomesh-lineage> (max 12 refs)
_ = snip
```

## Catalog (data products)

Fail-open discovery of governed data products. Tries mesh `/v1/catalog/*` then portal
`/v17|/v16` federation paths (404 → next; all fail → `Source=fail-open`).

```go
res := nc.ListCatalog(ctx, "") // optional query; "operational"|"knowledge"|"analytical" also sets mesh_layer=
fmt.Printf("source=%s products=%d\n", res.Source, len(res.Products))
fmt.Print(iomeshclient.FormatCatalog(res))

p, meta := nc.GetCatalogProduct(ctx, "engineering-github-events")
_ = p
_ = meta // Source mesh|portal|fail-open; Detail is path or error note
```

## Security

- Report vulnerabilities **privately**: [SECURITY.md](SECURITY.md) (GitHub Security Advisory or security@iome.sh).  
  Do **not** open public issues for exploits.
- Do **not** commit API tokens, broker URLs with credentials, or customer payloads into issues/PRs.
- Prefer short-lived bearer tokens (`WithBearerToken`) and tenant-scoped headers (`WithTenant` / `WithOrg` / `WithWorkspace`).
- Broker URLs must be absolute **`http`/`https`** (no `file://`, no embedded userinfo).
- Connector HMAC secrets must stay server-side; never embed partner secrets in mobile or browser clients.
- Treat `X-IOMesh-Tenant` / `X-IOMesh-Org` as an authorization boundary — **enforce server-side**.

## Versioning & support

- Semantic versioning (`vMAJOR.MINOR.PATCH`).
- Breaking changes only in major versions; see [CHANGELOG.md](CHANGELOG.md) and [RELEASING.md](RELEASING.md).
- Supported Go versions: last two stable releases (CI matrix).
- Help channels: [SUPPORT.md](SUPPORT.md).

## Development

```bash
go test ./...
go test -race ./...
golangci-lint run ./...   # if installed
```

This repository is **pure client code** — no private platform dependencies. Unit tests use `httptest` and local helpers. Live broker integration belongs in your environment or private test harnesses, not in this public tree.

**Public naming:** packages, env vars, and wire headers use `iomesh` / `IOMESH_*` / `X-IOMesh-*`. Internal monorepo codenames are not part of this SDK.

Process docs: [CONTRIBUTING](CONTRIBUTING.md) · [SUPPORT](SUPPORT.md) · [RELEASING](RELEASING.md) · [docs/OPEN_SOURCE_AUDIT.md](docs/OPEN_SOURCE_AUDIT.md).

## Related

| Link | Role |
|------|------|
| [iome.sh](https://iome.sh) | Product / marketing site & documentation |
| [iomesh-tui](https://github.com/iome-sh/iomesh-tui) | Agent edge TUI (`/memory`, integrations, pull) |
| [iomesh-memory-mcp](https://github.com/iome-sh/iomesh-memory-mcp) | Public edge Memory MCP host (local palace; dual_write OFF; not Memory GA) |
| [memory](https://github.com/iome-sh/memory) | Public palace kernel (Go module; not imported by this SDK) |
| [iomesh-client-sdk-python](https://github.com/iome-sh/iomesh-client-sdk-python) | Official Python client (see below) |
| *Upcoming* | `iomesh-client-sdk-ts`, … |

### Also available (other languages)

**[Python SDK](https://github.com/iome-sh/iomesh-client-sdk-python)** — official MIT peer. Both this Go client and the Python client are **Beta** / pre-1.0. Python **v0.1** covers the **core HTTP plane** (connect, tenant/org/workspace headers, publish, pull consumers, health). **v0.2** expands beyond core HTTP toward **KV · memory (dual_write OFF default) · connectorsdk HMAC**. **v0.3** expands toward **Kafka Produce subset · wait_ready · memory related/ops_digest**. **v0.4** adds residual polish toward **catalog · evaluate_policy**. **v0.5** residual polish toward **query_context · format helpers · connection_status**. **v0.6** residual polish toward **KV/msg format helpers**. **v0.7** residual polish toward **emit_dept_event · emit_llm_call** (heartbeat/pulse). **v0.8** residual polish toward **register_processor · list_live_views**. **v0.9** residual polish toward **request_memory_recall · connect_from_env**. **v0.10** residual polish toward **1.0 readiness bar docs · py.typed · pull_loop example** — still **Beta**, still **not** the full Go surface · dual_write OFF elsewhere · not Memory GA · not invent 1.0 GA. Do not invent GA, language parity, or liveview GA.

## License

[MIT](LICENSE) © 2026 [IOMesh Technology Ltd.](https://iome.sh) — see also [NOTICE](NOTICE).

**Maintained by** [IOMesh Technology Ltd.](https://iome.sh) · Product: [iome.sh](https://iome.sh) · Support: [SUPPORT.md](SUPPORT.md)
