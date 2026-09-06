# GitHub stream read

Replay messages from a **GitHub-ingested** stream the caller is allowed to see, using `ListStreamMessages`.

## What this shows

1. Connect to a broker you run or subscribe to
2. `GET /v1/streams/{name}/messages` via `ListStreamMessages`
3. Print seq / subject / payload size for GitHub-ingested rows

## Honesty

- Stream replay is **not** an org-health or heart-rate API
- Slack and PagerDuty are **not** live pulses in this example
- Chat is **not** the record
- Empty list is honest: no signed GitHub event yet · catalog ≠ Connected
- Offline tests ≠ live APPLY · Beta / pre-1.0 · dual_write OFF
- Omit-org on a shared stream can mix orgs (broker fail-open). Prefer `IOMESH_ORG` + `IOMESH_REQUIRE_ORG=1`

## Run

```bash
export IOMESH_URL=http://127.0.0.1:8422
export IOMESH_ORG=org_example          # X-IOMesh-Org on ListStreamMessages
export IOMESH_REQUIRE_ORG=1            # prefer fail-closed on shared github streams
go run ./examples/github-stream-read
```

`IOMESH_ORG` maps to `X-IOMesh-Org`. Omitting it on a shared stream (`github`, `OPERATIONAL_EVENTS`) can mix orgs on fail-open brokers. Prefer `IOMESH_REQUIRE_ORG=1` so the client errors before replay when org is empty. The library does not invent a default org.

| Env | Default | Notes |
|-----|---------|--------|
| `IOMESH_URL` | `http://127.0.0.1:8422` | broker base |
| `IOMESH_TENANT` | `dept.engineering` | `X-IOMesh-Tenant` |
| `IOMESH_ORG` | empty | `X-IOMesh-Org`; set for hosted isolation / N=2 shared streams |
| `IOMESH_REQUIRE_ORG` | off | `1`/`true`/`yes`/`on` fail-closes catalog/consume when org is empty |
| `IOMESH_WORKSPACE` / `IOMESH_DEPARTMENT` | empty | optional `X-IOMesh-*` identity headers (department omitted when empty) |
| `IOMESH_STREAM` | `OPERATIONAL_EVENTS` | GitHub-ingested durable stream |
| `IOMESH_LIMIT` | `50` | replay page size (cap 1000) |

## Related

- Package API: [`ListStreamMessages`](../../README.md#streams)
- Durable pull smoke: [`examples/pull-loop`](../pull-loop/)
