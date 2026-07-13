# I/O Mesh Client SDK for Go

[![CI](https://github.com/iome-sh/iomesh-client-sdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/iome-sh/iomesh-client-sdk-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/iome-sh/iomesh-client-sdk-go.svg)](https://pkg.go.dev/github.com/iome-sh/iomesh-client-sdk-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Official **Go client SDK** for the [I/O Mesh](https://iome.sh) broker and connector platform.

| Capability | Package |
|------------|---------|
| HTTP publish / pull subscribe / streams / KV / memory | [`aionclient`](./aionclient) |
| Partner webhook HMAC + observation envelopes | [`connectorsdk`](./connectorsdk) |
| Kafka protocol (Produce subset) | [`kafka`](./kafka) · via `aionclient.KafkaClient` |
| Shared envelope + CUID helpers | [`envelope`](./envelope) · [`cuid`](./cuid) |

> **Module path:** `github.com/iome-sh/iomesh-client-sdk-go`  
> **Package `aionclient`:** stable Go import path for the mesh HTTP client API (name retained for compatibility).

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

	"github.com/iome-sh/iomesh-client-sdk-go/aionclient"
)

func main() {
	nc, err := aionclient.Connect(
		aionclient.Options{URL: "http://127.0.0.1:8422"},
		aionclient.WithTenant("dept.engineering"),
		aionclient.WithOrg("acme-org"),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	if err := nc.CreateStream(ctx, aionclient.StreamConfig{
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

See [`examples/connector-sdk-template/`](examples/connector-sdk-template/) for a full webhook adapter.

## Kafka Produce

```go
kc := aionclient.NewKafkaClient("127.0.0.1:9423")
defer kc.Close()

offset, err := kc.Produce(ctx, "mesh.finance.events", 0, []byte("key"), []byte(`{"event_id":"evt-1"}`))
```

## Security

- Report vulnerabilities privately: see [SECURITY.md](SECURITY.md).
- Do **not** commit API tokens, broker URLs with credentials, or customer payloads into issues/PRs.
- Prefer short-lived bearer tokens (`WithBearerToken`) and tenant-scoped headers (`WithTenant` / `WithOrg`).
- Connector HMAC secrets must stay server-side; never embed partner secrets in mobile or browser clients.

## Versioning & support

- Semantic versioning (`vMAJOR.MINOR.PATCH`).
- Breaking changes only in major versions; see [CHANGELOG.md](CHANGELOG.md) and [RELEASING.md](RELEASING.md).
- Supported Go versions: last two stable releases (CI matrix).

## Development

```bash
go test ./...
go test -race ./...
golangci-lint run ./...   # if installed
```

This repository is **pure client code** — no private platform dependencies. Unit tests use `httptest` and local helpers. Live broker integration belongs in your environment or private test harnesses, not in this public tree.

## Related

| Link | Role |
|------|------|
| [iome.sh](https://iome.sh) | Product & documentation |
| [`iome-sh/iomesh`](https://github.com/iome-sh/iomesh) | Public marketing site |
| *Upcoming* | `iomesh-client-sdk-ts`, `iomesh-client-sdk-python`, … |

## License

[MIT](LICENSE) © 2026 [IOMesh Technology Ltd.](https://iome.sh)
