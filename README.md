# I/O Mesh Client SDK for Go

[![CI](https://github.com/iome-sh/iomesh-client-sdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/iome-sh/iomesh-client-sdk-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/iome-sh/iomesh-client-sdk-go.svg)](https://pkg.go.dev/github.com/iome-sh/iomesh-client-sdk-go)
[![GitHub release](https://img.shields.io/github/v/release/iome-sh/iomesh-client-sdk-go)](https://github.com/iome-sh/iomesh-client-sdk-go/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/status-Beta%20pre--1.0-yellow.svg)](#status)

Official **Go client** for the [I/O Mesh](https://iome.sh) broker: HTTP publish/pull, streams, KV, and a Kafka Produce subset. **MIT**. **Beta / pre-1.0**. From [IOMesh](https://iome.sh) (**IOMesh Technology Ltd.**).

Connectors and services publish **organizational heartbeats** (ops **pulse**) on `dept.*` streams; agents and workers consume them with durable pull.

This repository is **edge client code** only — not hosted control-plane access.

## Contents

- [Status](#status)
- [Install](#install)
- [Quick start](#quick-start)
- [Environment](#environment)
- [Capabilities](#capabilities)
- [API](#api)
- [Examples](#examples)
- [License](#license)

## Status

Public OSS **[v0.71.0](https://github.com/iome-sh/iomesh-client-sdk-go/releases/tag/v0.71.0)** — **Beta / pre-1.0**. APIs may change before 1.0. See [CHANGELOG.md](CHANGELOG.md).

- **MIT edge client** — not free mesh control-plane access.
- **Catalog list ≠ Connected.** `ListCatalog` is data-product discovery (Knowledge stays Beta), not a connector install or OAuth wrap.
- **Memory helpers** talk to a **local sidecar**. `DualWriteMemoryTurn` is async-only unless you set `Sync: true`. A mesh-broker URL may 404 retrieve/ingest.
- `ConsumerNack` / `DeleteConsumer` are client wrappers; the serving broker may 404 until those routes exist (create/fetch/ack are the served durable-pull set).

Module `github.com/iome-sh/iomesh-client-sdk-go` · package `iomeshclient` · User-Agent `iomesh-client-sdk-go/<Version>` (override with `WithUserAgent`).

## Requirements

- Go **1.27+** (module declares the toolchain used in CI)
- Network access to an I/O Mesh broker (or local foundation)

## Install

```bash
go get github.com/iome-sh/iomesh-client-sdk-go@latest
```

## Quick start

Connect, ensure a `dept.*` stream, and publish one organizational heartbeat. Needs a reachable broker.

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
		iomeshclient.WithOrg("org_example"), // local/dev placeholder; hosted: CP-minted org_+cuid2
		iomeshclient.WithDepartment("engineering"),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	_, err = nc.CreateStream(ctx, iomeshclient.StreamConfig{
		Name:     "EVENTS",
		Subjects: []string{"dept.engineering.events.>"},
	})
	if err != nil {
		log.Fatal(err)
	}

	ack, err := nc.Publish(ctx, "EVENTS", "dept.engineering.events.demo",
		[]byte(`{"hello":"mesh","kind":"org_heartbeat"}`))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("published seq=%d subject=%s partition=%d", ack.Seq, ack.Subject, ack.Partition)
}
```

Omit `WithWorkspace` / `IOMESH_WORKSPACE` so the broker binds the org **root-default**. This client never invents `workspaces[0]` (creation-order first row is not the root bind). Hosted workspace ids are `ws_`+cuid2; `ws_default` is not a minted id.

Runnable framing (publish + optional pull): [`examples/org-heartbeat-publish/`](examples/org-heartbeat-publish/). Durable pull loop: [`examples/pull-loop/`](examples/pull-loop/).

## Environment

`ConnectFromEnv(nil)` reads process env. `IOMESH_URL` is required; the rest are optional. No network I/O on connect.

| Variable | Header / effect |
|----------|-----------------|
| `IOMESH_URL` | Broker base (`http`/`https`) |
| `IOMESH_TENANT` | `X-IOMesh-Tenant` |
| `IOMESH_ORG` | `X-IOMesh-Org` — hosted: CP-minted `org_`+cuid2; `org_example` is a local/dev placeholder |
| `IOMESH_WORKSPACE` | `X-IOMesh-Workspace` — omit blank = broker root-default |
| `IOMESH_DEPARTMENT` | `X-IOMesh-Department` (omit when empty) |
| `IOMESH_BEARER_TOKEN` or `IOMESH_TOKEN` | `Authorization: Bearer` (`BEARER_TOKEN` wins if both set) |
| `IOMESH_TIMEOUT` | Request timeout in seconds (float; default 30) |
| `IOMESH_REQUIRE_ORG` | `1`/`true`/`yes`/`on` — fail-closed catalog/consume when org is empty |

```go
nc, err := iomeshclient.ConnectFromEnv(nil)
```

Hosted brokers isolate catalog and durable pull by `X-IOMesh-Org`. Omitting it can mix shared-stream reads (or the broker may reject). Local/dev brokers still fail-open when org is empty. The library does **not** invent a default org. `cuid.NewOrgID` / `cuid.NewWorkspaceID` mint the **shape** only — they do not register an org or workspace.

## Capabilities

| Capability | Package | Notes |
|------------|---------|--------|
| HTTP publish / pull / streams / KV / memory | [`iomeshclient`](./iomeshclient) | Org heartbeats on `dept.*` |
| Partner webhook HMAC + observation envelopes | [`connectorsdk`](./connectorsdk) | Local HMAC; not OAuth |
| Kafka Produce subset | [`kafka`](./kafka) · `iomeshclient.KafkaClient` | Produce-only |
| Shared envelope + CUID helpers | [`envelope`](./envelope) · [`cuid`](./cuid) | `org_` / `ws_` + cuid2 shapes |

## API

Full godoc: [pkg.go.dev/github.com/iome-sh/iomesh-client-sdk-go](https://pkg.go.dev/github.com/iome-sh/iomesh-client-sdk-go).

### Connector SDK (HMAC + envelope)

```go
import "github.com/iome-sh/iomesh-client-sdk-go/connectorsdk"

payload, err := connectorsdk.NormalizeEnvelope(
	"acme-crm", "engineering", "acme-crm", "evt-42", "contact.created",
	json.RawMessage(`{"email":"user@example.com"}`),
)
```

See [`examples/connector-sdk-template/`](examples/connector-sdk-template/) (`IOMESH_URL`, `IOMESH_ORG`, optional `IOMESH_INSTALL_ID`). HMAC verify ≠ OAuth.

### Kafka Produce

```go
kc := iomeshclient.NewKafkaClient("127.0.0.1:9423")
defer kc.Close()

offset, err := kc.Produce(ctx, "mesh.finance.events", 0, []byte("key"), []byte(`{"event_id":"evt-1"}`))
```

### Streams

| API | Path | Notes |
|-----|------|--------|
| `CreateStream` / `EnsureStream` | `POST /v1/streams` | `*StreamInfo`; 409 → success + best-effort GET |
| `ListStreams` / `GetStream` | `GET /v1/streams…` | Explicit discovery; non-2xx → `*APIError`. Hosted list is org-scoped when `X-IOMesh-Org` is set |
| `DeleteStream` | `DELETE /v1/streams/{name}` | Destructive; 404 → `*APIError` |
| `ListStreamMessages` | `GET …/messages` | Replay range (`from_seq` / `to_seq` / `limit`) |
| `CreateConsumer` / `EnsureConsumer` | `POST …/consumers` | 409 → Stream/Name only |
| `DeleteConsumer` | `DELETE …/consumers/{name}` | Client wrapper; broker may 404 |
| `ConsumerFetch` / `ConsumerAck` / `ConsumerNack` | `POST …/fetch\|ack\|nack` | **Ack is served**; nack may 404 |
| `Publish` / `PullSubscribe` | stream / consumer | `FetchContext` / `AckContext`; default long-poll `DefaultFetchMaxWait` (5s) |
| `FormatStreams` / `FormatMsg` / … | — | Operator string views (no network) |
| `Pub` | `POST /v1/pub` | Ephemeral fire-and-forget |

```go
streams, err := nc.ListStreams(ctx)
fmt.Print(iomeshclient.FormatStreams(streams))

sub, err := nc.PullSubscribe(ctx, iomeshclient.PullSubscribeConfig{
	Stream: "EVENTS", Consumer: "worker-1", Filter: "dept.events.>",
})
batch, err := sub.FetchContext(ctx, 10, iomeshclient.MaxWait(2*time.Second))
fmt.Print(iomeshclient.FormatMsgs(batch))
_ = sub.AckContext(ctx /* seqs... */)
```

### KV

| API | Path | Notes |
|-----|------|--------|
| `CreateBucket` / `EnsureBucket` | `POST /v1/kv/{name}` | 409 → name-only `*BucketInfo` |
| `Put` / `Get` / `Delete` | `/v1/kv/{bucket}/{key}` | Put returns `*PutResult`; Get returns `*KVEntry` |
| `ListKeys` | `GET /v1/kv/{bucket}?prefix=` | Optional prefix |
| `FormatBucketInfo` / `FormatKVEntry` / … | — | Operator string views |

```go
_, _ = nc.EnsureBucket(ctx, "agent-state", iomeshclient.CreateBucketConfig{History: 5})
put, _ := nc.Put(ctx, "agent-state", "worker-1.checkpoint", []byte("seq=42"))
fmt.Print(iomeshclient.FormatPutResult(*put))
```

### Memory

Retrieve and ingest helpers talk to a **memory sidecar** (or a gateway that routes those paths). Durable stream ingest is the default; set `Sync: true` for an optional sidecar write.

| API | Path | Notes |
|-----|------|--------|
| `PublishMemoryIngest` | `MEMORY_INGEST` publish | Async durable stream |
| `DualWriteMemoryTurn` | async + optional sync | Stream first; `Sync: false` by default |
| `RequestMemoryRecall` / `RequestMemoryRecallFull` | `MEMORY_RPC` publish | Async; Full adds `session_id` |
| `RetrieveMemory` | `POST /v1` then `/v5/memory/retrieve` | Sidecar HTTP; mesh-broker URL typically 404s |
| `IngestMemoryTurn` | `POST /v1` then `/v5/memory/ingest` | Optional sidecar write |

```go
res, err := nc.DualWriteMemoryTurn(ctx, "dept.research", iomeshclient.MemoryEnvelope{
	Role: "user", Content: "decision notes", SessionID: "sess-1", SessionSeq: 1,
}, iomeshclient.DualWriteMemoryOptions{}) // Sync: false
```

This SDK is the **mesh/platform HTTP** client. It does not import `github.com/iome-sh/memory` and does not attach MCP stdio.

| Plane | Package | Role |
|-------|---------|------|
| Local edge palace | [`iomesh-memory-mcp`](https://github.com/iome-sh/iomesh-memory-mcp) + [`memory`](https://github.com/iome-sh/memory) | Customer-local MCP host + palace kernel |
| Mesh / platform HTTP | this SDK | Broker/gateway and optional sidecar HTTP |
| Private control plane | unpublished | Not a public dependency of this SDK |

```bash
go install github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main
go get github.com/iome-sh/memory@main
```

### Metering

`EmitDeptEvent` / `EmitLLMCall` publish structured org-tool heartbeats on stream `dept`.

```go
ack, err := nc.EmitLLMCall(ctx, iomeshclient.LLMCallEvent{
	Tenant: "dept.research", SessionID: "sess-1",
	Model: "deepseek-v4-flash", TotalTokens: 120, EstUSD: 0.002,
})
// POST /v1/streams/dept/publish  subject=dept.agent.llm_call
```

Stage smoke (mesh + optional sidecar): `go run ./examples/memory-metering-dogfood`.

### Diagnostics, policy, context

```go
fmt.Println(iomeshclient.Version)
_ = nc.Health(ctx)
_ = nc.Ready(ctx)
st := nc.ConnectionStatus(ctx)
fmt.Print(iomeshclient.FormatConnectionStatus(st))
_ = nc.WaitReady(ctx, iomeshclient.WaitReadyOptions{Interval: 500 * time.Millisecond})

dec := nc.EvaluatePolicy(ctx, iomeshclient.PolicyInput{
	Tool: "run_shell", Mode: iomeshclient.PolicyEnforce,
})
if dec.ShouldBlockTool() { /* mesh deny under enforce */ }

snip := nc.ContextSnippet(ctx, ".", "incidents last hour")
_ = snip
```

Policy and context are fail-open on transport / 404 so a missing plane does not block agent DX. Enforce only blocks when mesh explicitly denies.

### Catalog

Fail-open discovery of governed **data products**. Tries mesh `/v1/catalog/*` then portal `/v17|/v16` (404 → next).

```go
res := nc.ListCatalog(ctx, "")
fmt.Print(iomeshclient.FormatCatalog(res))
```

## Examples

```bash
export IOMESH_URL=http://127.0.0.1:8422
export IOMESH_ORG=org_example   # local/dev placeholder; hosted: CP-minted org_+cuid2
go run ./examples/org-heartbeat-publish
IOMESH_PULL=1 go run ./examples/org-heartbeat-publish

export IOMESH_STREAM=EVENTS
export IOMESH_CONSUMER=sdk-pull-loop
go run ./examples/pull-loop
```

| Example | What it shows |
|---------|----------------|
| [`examples/org-heartbeat-publish/`](examples/org-heartbeat-publish/) | Publish + optional pull of an org heartbeat |
| [`examples/pull-loop/`](examples/pull-loop/) | Multi-cycle durable fetch/ack (optional ensure/publish/strict) |
| [`examples/github-stream-read/`](examples/github-stream-read/) | Replay a GitHub-ingested stream via `ListStreamMessages` |
| [`examples/connector-sdk-template/`](examples/connector-sdk-template/) | Webhook HMAC + envelope + events URL |
| [`examples/memory-metering-dogfood/`](examples/memory-metering-dogfood/) | Metering pulse + optional sidecar memory |

Offline stage smoke is not a production rollout.

## Security

- Report vulnerabilities **privately**: [SECURITY.md](SECURITY.md). Do **not** open public issues for exploits.
- Do **not** commit API tokens or customer payloads into issues/PRs.
- Prefer short-lived bearer tokens (`WithBearerToken`) and tenant-scoped headers.
- Broker URLs must be absolute **`http`/`https`** (no `file://`, no embedded userinfo).
- Connector HMAC secrets stay server-side.
- Treat `X-IOMesh-Tenant` / `X-IOMesh-Org` / `X-IOMesh-Department` as an authorization boundary — **enforce server-side**.

## Versioning & support

- Semantic versioning (`vMAJOR.MINOR.PATCH`).
- Breaking changes only in major versions; see [CHANGELOG.md](CHANGELOG.md) and [RELEASING.md](RELEASING.md).
- Supported Go versions: last two stable releases (CI matrix).
- Help: [SUPPORT.md](SUPPORT.md).

## Development

```bash
go test ./...
go test -race ./...
golangci-lint run ./...   # if installed
```

This repository is **pure client code** — no private platform dependencies. Unit tests use `httptest`. Live broker integration belongs in your environment.

**Public naming:** packages, env vars, and wire headers use `iomesh` / `IOMESH_*` / `X-IOMesh-*`.

Process docs: [CONTRIBUTING](CONTRIBUTING.md) · [SUPPORT](SUPPORT.md) · [RELEASING](RELEASING.md) · [docs/OPEN_SOURCE_AUDIT.md](docs/OPEN_SOURCE_AUDIT.md).

## Related

| Link | Role |
|------|------|
| [iome.sh](https://iome.sh) | Product home |
| [pkg.go.dev](https://pkg.go.dev/github.com/iome-sh/iomesh-client-sdk-go) | API reference |
| [iomesh-tui](https://github.com/iome-sh/iomesh-tui) | Agent edge TUI |
| [iomesh-memory-mcp](https://github.com/iome-sh/iomesh-memory-mcp) | Public edge Memory MCP host |
| [memory](https://github.com/iome-sh/memory) | Public palace kernel (not imported by this SDK) |
| [iomesh-client-sdk-python](https://github.com/iome-sh/iomesh-client-sdk-python) | Official Python client (subset of this surface; also Beta / pre-1.0; not live PyPI) |

## License

[MIT](LICENSE) © 2026 [IOMesh Technology Ltd.](https://iome.sh) — see also [NOTICE](NOTICE).

**Maintained by** [IOMesh Technology Ltd.](https://iome.sh) · Product: [iome.sh](https://iome.sh) · Support: [SUPPORT.md](SUPPORT.md)
