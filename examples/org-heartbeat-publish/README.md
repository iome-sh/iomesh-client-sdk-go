# Org heartbeat publish / pull

Lightweight framing for **organizational heartbeats** (ops **pulse**) on `dept.*` streams using the public Go SDK.

## What this shows

1. **Publish** — `EnsureStream` + `Publish` a single org heartbeat under `dept.<tenant>.events.*`
2. **Structured pulse** — optional `EmitDeptEvent` (`dept.agent.org_heartbeat`) for metering-style org-tool events
3. **Pull** (opt-in) — `IOMESH_PULL=1` runs `PullSubscribe` + one `FetchContext` so agents can consume the same subjects

## Honesty

- Public lexicon: **heartbeat / pulse** only
- Offline stage smoke **≠** live APPLY
- Surfaces are **Beta** / pre-1.0 — no invent GA
- Memory dual_write defaults **OFF** (local-primary async); this example does not enable sync palace
- MIT edge client only — not free mesh control-plane access

## Run

```bash
export IOMESH_URL=http://127.0.0.1:8422
export IOMESH_ORG=org_example
go run ./examples/org-heartbeat-publish

# optional durable pull of the same subject tree
IOMESH_PULL=1 go run ./examples/org-heartbeat-publish
```

`IOMESH_ORG` maps to `X-IOMesh-Org` on publish/fetch/ack. Omitting it can mix shared-stream reads on fail-open brokers. Set `IOMESH_REQUIRE_ORG=1` so the client errors before catalog/consume when org is empty.

| Env | Default | Notes |
|-----|---------|--------|
| `IOMESH_URL` | `http://127.0.0.1:8422` | broker base |
| `IOMESH_TENANT` | `dept.engineering` | `X-IOMesh-Tenant` |
| `IOMESH_ORG` | empty | `X-IOMesh-Org`; set for hosted isolation |
| `IOMESH_REQUIRE_ORG` | off | `1`/`true`/`yes`/`on` fail-closes catalog/consume when org is empty |
| `IOMESH_WORKSPACE` | empty | `X-IOMesh-Workspace` |
| `IOMESH_STREAM` | `EVENTS` | durable stream name |
| `IOMESH_SUBJECT` | `<tenant>.events.org-heartbeat` | publish subject |
| `IOMESH_PULL` | off | set `1` for one fetch cycle |
| `IOMESH_CONSUMER` | `sdk-org-heartbeat` | when pull enabled |

## Related

- Multi-cycle stage smoke with `SUMMARY` / `RESULT` scrapers: [`../pull-loop/`](../pull-loop/)
- Memory + metering dogfood (dual_write sync only when sidecar URL differs; optional `IOMESH_PREFER_SHORTER_HOPS` for related hop ranking — omit = kernel default true · multi-hop lite ≠ full graph RAG): [`../memory-metering-dogfood/`](../memory-metering-dogfood/)
- Main SDK README quick start: publish/pull org heartbeats framing
