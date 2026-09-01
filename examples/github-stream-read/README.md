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

## Run

```bash
export IOMESH_URL=http://127.0.0.1:8422
go run ./examples/github-stream-read
```

| Env | Default | Notes |
|-----|---------|--------|
| `IOMESH_URL` | `http://127.0.0.1:8422` | broker base |
| `IOMESH_TENANT` | `dept.engineering` | `X-IOMesh-Tenant` |
| `IOMESH_STREAM` | `OPERATIONAL_EVENTS` | GitHub-ingested durable stream |
| `IOMESH_LIMIT` | `50` | replay page size (cap 1000) |

## Related

- Package API: [`ListStreamMessages`](../../README.md#streams)
- Durable pull smoke: [`examples/pull-loop`](../pull-loop/)
