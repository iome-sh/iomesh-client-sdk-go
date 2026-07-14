# Security Policy

## Supported versions

| Version | Supported |
|---------|-----------|
| `v0.1.x` | ✅ |
| `< v0.1` | ❌ |

## Reporting a vulnerability

**Please do not open public GitHub issues for security vulnerabilities.**

Email **security@iome.sh** (or the security contact listed at [iome.sh](https://iome.sh)) with:

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

Out of scope:

- Vulnerabilities in a self-hosted I/O Mesh broker deployment (report via platform support)  
- Social engineering / phishing  
- Issues only present when using untrusted third-party forks  

## Hardening expectations for consumers

- Pin module versions (`go get …@vX.Y.Z` + committed `go.sum`)  
- Rotate broker tokens and connector secrets regularly  
- Treat `X-IOMesh-Tenant` / org headers as authorization boundary — enforce server-side  
- Run `go mod verify` in CI  

## Supply chain

- Releases are Git tags on `main` (`v*`)  
- CI runs tests on every PR and on `main`  
- Dependabot (or equivalent) updates Go modules and GitHub Actions  
