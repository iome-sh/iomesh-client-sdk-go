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
> **Status:** public OSS **v0.6.x** (pre-1.0). Memory M2/M3 + multi-tenant headers + dual-write/metering helpers aligned with [iomesh-tui](https://github.com/iome-sh/iomesh-tui).  
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
	if err := nc.CreateStream(ctx, iomeshclient.StreamConfig{
		Name:     "EVENTS",
		Subjects: []string{"dept.engineering.events.>"},
	}); err != nil {
		log.Fatal(err)
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

## Diagnostics

```go
fmt.Println(iomeshclient.Version) // e.g. "0.6.0"
// All requests send: User-Agent: iomesh-client-sdk-go/0.6.0
// Override: iomeshclient.WithUserAgent("my-service/1.2.3")
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
