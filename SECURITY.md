# Security Policy

## Supported versions

| Version | Supported |
|---------|-----------|
| `v0.43.x` (latest minor on `main`) | ✅ security fixes |
| `v0.42.x` | best-effort |
| `v0.41.x` | best-effort |
| `v0.24.x` | best-effort |
| `v0.23.x` | best-effort |
| `v0.22.x` | best-effort |
| `v0.21.x` | best-effort |
| `v0.20.x` | best-effort |
| `v0.19.x` | best-effort |
| `v0.18.x` | best-effort |
| `v0.17.x` | best-effort |
| `v0.16.x` | best-effort |
| `v0.15.x` | best-effort |
| `v0.14.x` | best-effort |
| `v0.13.x` | best-effort |
| `v0.12.x` | best-effort |
| `v0.11.x` | best-effort |
| `v0.10.x` | best-effort |
| `v0.9.x` | best-effort |
| `v0.7.x` | best-effort |
| `v0.6.x` | best-effort |
| `v0.5.x` | best-effort |
| `v0.4.x` | best-effort |
| `v0.3.x` | best-effort |
| `v0.2.x` | best-effort |
| `v0.1.x` | best-effort until EOL notice |
| pre-release / untagged | best-effort |

## Reporting a vulnerability

**Please do not open public GitHub issues for security vulnerabilities.**

Preferred channels (in order):

1. **GitHub Security Advisory** (private) — Security → Advisories → Report a vulnerability  
2. Email **security@iome.sh** (or the security contact listed at [iome.sh](https://iome.sh))

Include:

1. Description of the issue and impact  
2. Steps to reproduce (PoC if available)  
3. Affected module version / commit  
4. Whether you plan coordinated disclosure  

We aim to acknowledge within **2 business days** and provide a status update within **7 days**.

## Scope

In scope for this repository:

- Client credential handling and HTTP auth headers  
- Connector HMAC verification helpers  
- Unsafe deserialization or injection via SDK helpers  
- Supply-chain issues in release artifacts (tags, `go.sum`)  
- Broker URL parsing / scheme handling  

Out of scope:

- Vulnerabilities in a self-hosted I/O Mesh broker deployment (report via platform support)  
- Social engineering / phishing  
- Issues only present when using untrusted third-party forks  

## Hardening expectations for consumers

- Pin module versions (`go get …@vX.Y.Z` + committed `go.sum`)  
- Rotate broker tokens and connector secrets regularly  
- Use **HTTPS** broker URLs in production  
- Prefer `WithBearerToken` — never put credentials in the broker URL  
- Treat `X-IOMesh-Tenant` / org headers as an authorization boundary — **enforce server-side**  
- Keep HMAC secrets server-side only (never in mobile/browser clients)  
- Run `go mod verify` in CI  

## Supply chain

- Releases are Git tags on `main` (`v*`)  
- CI runs tests, race, vet, govulncheck, and golangci-lint on every PR and on `main`  
- Dependabot updates Go modules and GitHub Actions  

## Residual risks (honest)

| Risk | Residual |
|------|----------|
| Misconfigured broker trust | Client will talk to any `http(s)` host you pass; validate endpoints in your app |
| Tenant header spoofing | Headers are not crypto auth — broker must authorize |
| Long-lived bearer tokens | Caller responsibility to refresh/rotate |
