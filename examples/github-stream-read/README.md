# GitHub stream read

Replay messages from a **GitHub-ingested** stream the caller is allowed to see, using `ListStreamMessages`.

## What this shows

1. Connect to a broker you run or subscribe to
2. `GET /v1/streams/{name}/messages` via `ListStreamMessages`
3. Print seq / subject / payload size for GitHub-ingested rows

## Notes

- Stream replay is **not** an org-health or heart-rate API
- Slack and PagerDuty are **not** live pulses in this example
- Chat is **not** the record
- An empty list means no signed GitHub event yet; catalog list is not a Connected install
- Offline tests are not a live apply; surfaces are Beta / pre-1.0
- Omit-org on a shared stream can mix orgs (broker fail-open). Prefer `IOMESH_ORG` + `IOMESH_REQUIRE_ORG=1`

## Run

```bash
export IOMESH_URL=http://127.0.0.1:8422
export IOMESH_ORG=org_example          # local/dev placeholder — hosted: CP-minted org_+cuid2
# omit IOMESH_WORKSPACE — broker root-default; never invent workspaces[0] / ws_default
export IOMESH_REQUIRE_ORG=1            # prefer fail-closed on shared github streams
go run ./examples/github-stream-read
```

`IOMESH_ORG` maps to `X-IOMesh-Org`. Hosted brokers expect a control-plane minted `org_`+cuid2; `org_example` is a local/dev placeholder, not a minted id. Omitting org on a shared stream (`github`, `OPERATIONAL_EVENTS`) can mix orgs on fail-open brokers. Prefer `IOMESH_REQUIRE_ORG=1` so the client errors before replay when org is empty. The library does not invent a default org. Omit blank `IOMESH_WORKSPACE` so the broker uses its root-default workspace (never invent `workspaces[0]`).

| Env | Default | Notes |
|-----|---------|--------|
| `IOMESH_URL` | `http://127.0.0.1:8422` | broker base |
| `IOMESH_TENANT` | `dept.engineering` | `X-IOMesh-Tenant` |
| `IOMESH_ORG` | empty | `X-IOMesh-Org`; hosted: CP-minted `org_`+cuid2; `org_example` is a local/dev placeholder |
| `IOMESH_REQUIRE_ORG` | off | `1`/`true`/`yes`/`on` fail-closes catalog/consume when org is empty |
| `IOMESH_WORKSPACE` | empty | omit = broker root-default; never invent `workspaces[0]`; hosted: `ws_`+cuid2 |
| `IOMESH_DEPARTMENT` | empty | optional `X-IOMesh-Department` (omit when empty) |
| `IOMESH_STREAM` | `OPERATIONAL_EVENTS` | GitHub-ingested durable stream |
| `IOMESH_LIMIT` | `50` | replay page size (cap 1000) |

## Related

- Package API: [`ListStreamMessages`](../../README.md#streams)
- Durable pull smoke: [`examples/pull-loop`](../pull-loop/)
