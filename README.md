# I/O Mesh Client SDK for Go

[![CI](https://github.com/iome-sh/iomesh-client-sdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/iome-sh/iomesh-client-sdk-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/iome-sh/iomesh-client-sdk-go.svg)](https://pkg.go.dev/github.com/iome-sh/iomesh-client-sdk-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Official **Go client SDK** for the [I/O Mesh](https://iome.sh) broker and connector platform.

Official open-source tooling from [IOMesh](https://iome.sh) (**IOMesh Technology Ltd.**).

| Capability | Package |
|------------|---------|
| HTTP publish / pull subscribe / streams / KV / memory | [`iomeshclient`](./iomeshclient) |
| Partner webhook HMAC + observation envelopes | [`connectorsdk`](./connectorsdk) |
| Kafka protocol (Produce subset) | [`kafka`](./kafka) · via `iomeshclient.KafkaClient` |
| Shared envelope + CUID helpers | [`envelope`](./envelope) · [`cuid`](./cuid) |

> **Module path:** `github.com/iome-sh/iomesh-client-sdk-go`  
> **Package:** `iomeshclient`  
> **Env prefix:** `IOMESH_*`  
> **Wire headers:** `X-IOMesh-Tenant`, `X-IOMesh-Org`, `X-IOMesh-Workspace`, …  
> **Status:** public OSS **v0.26.x** (pre-1.0). Memory M2/M3 + multi-tenant headers + dual-write/metering + Health/Ready/WaitReady + catalog plane + EvaluatePolicy + QueryContext + ConnectionStatus + ListStreams/GetStream/DeleteStream/ListStreamMessages + CreateStream/EnsureStream `*StreamInfo` + FormatStreams/FormatStreamDetail + CreateConsumer/EnsureConsumer `*ConsumerInfo` + ConsumerFetch/ConsumerAck/ConsumerNack + PullSubscribe `FetchContext`/`AckContext`/`NackContext` + `DefaultFetchMaxWait` + FormatMsg/FormatMsgs/FormatConsumerInfo + KV CreateBucket/EnsureBucket `*BucketInfo` + Put `*PutResult` + FormatBucketInfo/FormatKVEntry/FormatKVKeys/FormatPutResult aligned with [iomesh-tui](https://github.com/iome-sh/iomesh-tui).  
> **User-Agent:** `iomesh-client-sdk-go/<Version>` (override with `WithUserAgent`).

## Requirements

- Go **1.22+** (module declares the toolchain used in CI)
- Network access to an I/O Mesh broker (or local foundation)

## Install

```bash
go get github.com/iome-sh/iomesh-client-sdk-go@latest
```

## Quick start — connect and publish

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

	ack, err := nc.Publish(ctx, "EVENTS", "dept.engineering.events.demo", []byte(`{"hello":"mesh"}`))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("published seq=%d subject=%s partition=%d", ack.Seq, ack.Subject, ack.Partition)
}
```

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
| `ConsumerFetch` / `ConsumerAck` / `ConsumerNack` | `POST …/fetch\|ack\|nack` | One-shot ops without holding a `Subscription`; path-escape stream/consumer; Fetch wires `Msg.Ack`/`Msg.Nack` via ephemeral sub |
| `Publish` / `PullSubscribe` | stream publish / consumer | `PullSubscribe` uses `CreateConsumer` then returns `*Subscription` with `ConsumerInfo()`; `FetchContext`/`AckContext`/`NackContext` (or `Fetch`/`Ack`/`Nack` → `context.Background()`); default long-poll `DefaultFetchMaxWait` (5s) / `MaxWait`; path segments escaped |
| `FormatMsg` / `FormatMsgs` / `FormatConsumerInfo` | — | Pure operator helpers for one message / batch / consumer detail (no network I/O) |
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
fmt.Print(iomeshclient.FormatStreamDetail(*info)) // multi-line detail

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

// PullSubscribe: CreateConsumer + subscription handle for Fetch/Ack/Nack
sub, err := nc.PullSubscribe(ctx, iomeshclient.PullSubscribeConfig{
	Stream: "EVENTS", Consumer: "worker-1", Filter: "dept.events.>",
})
if err != nil {
	log.Fatal(err)
}
fmt.Print(iomeshclient.FormatConsumerInfo(sub.ConsumerInfo()))

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
// Runnable stage smoke: examples/pull-loop (IOMESH_URL, optional IOMESH_ENSURE_STREAM / IOMESH_PUBLISH / IOMESH_ACK)

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
| `FormatBucketInfo` / `FormatKVEntry` / `FormatKVKeys` / `FormatPutResult` | — | Pure operator format helpers (no network I/O) |

```go
info, err := nc.EnsureBucket(ctx, "agent-state", iomeshclient.CreateBucketConfig{
	History: 5,
})
if err != nil {
	log.Fatal(err)
}
if info != nil {
	fmt.Print(iomeshclient.FormatBucketInfo(*info)) // multi-line bucket detail
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

## Memory (async streams + sync sidecar)

| API | Path | Notes |
|-----|------|--------|
| `PublishMemoryIngest` | `MEMORY_INGEST` publish | Async stream dual-write; temporal fields on `MemoryEnvelope` |
| `DualWriteMemoryTurn` | async + optional sync | Stream first; optional fail-open `IngestMemoryTurn` (sidecar) |
| `RequestMemoryRecall` / `RequestMemoryRecallFull` | `MEMORY_RPC` publish | Async; Full adds `session_id` correlation |
| `RetrieveMemory` | `POST /v1` then `/v5/memory/retrieve` | Sync hits; empty query OK if `session_id` set |
| `IngestMemoryTurn` | `POST /v1` then `/v5/memory/ingest` | Sync Palace turn write |

```go
// Sync retrieve (sidecar URL or gateway that routes /v1|/v5/memory/*)
hits, err := nc.RetrieveMemory(ctx, iomeshclient.MemoryRetrieveRequest{
	TenantID:  "dept.research",
	Query:     "lease rotation",
	SessionID: "dept.research.mesh-dogfood",
	Limit:     8,
})
// hits.Path is "/v1/memory/retrieve" or "/v5/memory/retrieve"

// Dual-write: durable stream + optional Palace sync (fail-open on sync)
mesh, _ := iomeshclient.Connect(iomeshclient.Options{URL: os.Getenv("IOMESH_URL")}, /* tenant/org… */)
palace, _ := iomeshclient.Connect(iomeshclient.Options{URL: os.Getenv("IOMESH_MEMORY_ENDPOINT")})
res, err := mesh.DualWriteMemoryTurn(ctx, "dept.research", iomeshclient.MemoryEnvelope{
	Role: "user", Content: "decision notes", SessionID: "sess-1", SessionSeq: 1,
}, iomeshclient.DualWriteMemoryOptions{Sync: true, SyncClient: palace})
// res.Async is PubAck; res.SyncErr is nil on Palace success (or set when fail-open)
```

The agent harness ([iomesh-tui](https://github.com/iome-sh/iomesh-tui)) mirrors these surfaces without depending on this module (lean public HTTP).
## Metering (dept streams)

```go
// Remote multi-tenant usage event for platform dashboards
ack, err := nc.EmitLLMCall(ctx, iomeshclient.LLMCallEvent{
	Tenant: "dept.research", SessionID: "sess-1",
	Model: "deepseek-v4-flash", TotalTokens: 120, EstUSD: 0.002,
})
// Wire: POST /v1/streams/dept/publish subject=dept.agent.llm_call
```

Stage smoke (mesh + optional memory sidecar):

```bash
export IOMESH_URL=http://127.0.0.1:8422
export IOMESH_MEMORY_ENDPOINT=http://127.0.0.1:8765  # warm plane
go run ./examples/memory-metering-dogfood
```

Pull consumer stage smoke (one fetch cycle; optional ensure/publish/ack):

```bash
export IOMESH_URL=http://127.0.0.1:8422
export IOMESH_STREAM=EVENTS
export IOMESH_CONSUMER=sdk-pull-loop
# export IOMESH_ENSURE_STREAM=1  # create stream with subject stream.>
# export IOMESH_PUBLISH=1        # publish one demo message before fetch
# export IOMESH_ACK=1            # ack fetched sequences
go run ./examples/pull-loop
```

See [`examples/pull-loop/`](examples/pull-loop/) for env flags (`IOMESH_BATCH`, `IOMESH_MAX_WAIT_MS`, `IOMESH_SUBJECT`, `IOMESH_PUBLISH`, …).

## Diagnostics

```go
fmt.Println(iomeshclient.Version) // e.g. "0.26.0"
// All requests send: User-Agent: iomesh-client-sdk-go/0.26.0
// Override: iomeshclient.WithUserAgent("my-service/1.2.3")

if err := nc.Health(ctx); err != nil { /* broker down */ }
if err := nc.Ready(ctx); err != nil { /* optional readiness path missing */ }

// One-shot identity + Health + Ready (fail-open; never panics). Both probes always run.
st := nc.ConnectionStatus(ctx)
fmt.Print(iomeshclient.FormatConnectionStatus(st))
// or: fmt.Print(iomeshclient.FormatConnectionStatusJSON(st))

// Poll until Ready (optional Health) or ctx deadline.
if err := nc.WaitReady(ctx, iomeshclient.WaitReadyOptions{
	Interval: 500 * time.Millisecond, // default when zero
	// RequireHealth: true,
}); err != nil { /* still not ready */ }

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
| *Upcoming* | `iomesh-client-sdk-ts`, `iomesh-client-sdk-python`, … |

## License

[MIT](LICENSE) © 2026 [IOMesh Technology Ltd.](https://iome.sh) — see also [NOTICE](NOTICE).

**Maintained by** [IOMesh Technology Ltd.](https://iome.sh) · Product: [iome.sh](https://iome.sh) · Support: [SUPPORT.md](SUPPORT.md)
