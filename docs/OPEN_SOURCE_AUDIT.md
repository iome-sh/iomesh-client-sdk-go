# Open-source readiness audit

Checklist for the public module `github.com/iome-sh/iomesh-client-sdk-go`.
Re-run before major releases.

## Security

| Check | Status |
|-------|--------|
| No committed API keys / private keys / `.env` secrets | Pass |
| No private monorepo `replace` in published `go.mod` | Pass |
| No private platform package imports | Pass (pure client) |
| Broker URL restricted to `http`/`https`; no URL userinfo | Pass (`validateBrokerURL`) |
| HMAC helpers use `hmac.Equal` (constant-time) | Pass |
| Bearer tokens via `WithBearerToken` (not URL credentials) | Pass |
| Vulnerability reporting path (advisory + security@iome.sh) | Pass |
| Secret scanning / push protection on GitHub | Enabled (org settings) |
| Residual risks documented | Pass (SECURITY.md) |

## Open-source process

| Artifact | Status |
|----------|--------|
| LICENSE (MIT) | Present |
| NOTICE | Present |
| CODE_OF_CONDUCT | Present |
| CONTRIBUTING | Present |
| SECURITY | Present |
| SUPPORT | Present |
| CHANGELOG | Present |
| RELEASING | Present |
| CI (test, race, vet, govulncheck, golangci-lint) | Present |
| Dependabot (gomod + actions) | Present |
| Issue templates + security contact | Present |
| PR template | Present |
| CODEOWNERS | Present |

## Consumer guidance (residual)

| Risk | Guidance |
|------|----------|
| Tenant/org headers are not authentication alone | Enforce authorization on the broker |
| HMAC secrets | Server-side only; never ship in mobile/browser |
| Long-lived bearer tokens | Prefer short-lived tokens; rotate regularly |
| Untrusted broker URL | Use HTTPS in production; pin versions |

## Maintainer actions

1. Keep private vulnerability reporting enabled  
2. Branch-protect `main` with required CI  
3. Tag releases per [RELEASING.md](../RELEASING.md)  
4. Never publish private stage URLs or customer tokens in issues/docs  
