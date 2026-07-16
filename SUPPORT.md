# Support

## How to get help

| Need | Where |
|------|--------|
| Usage questions / bugs | [GitHub Issues](https://github.com/iome-sh/iomesh-client-sdk-go/issues) (use templates) |
| Security vulnerability | Private [Security Advisory](https://github.com/iome-sh/iomesh-client-sdk-go/security/advisories/new) or **security@iome.sh** — see [SECURITY.md](SECURITY.md) |
| Product / broker platform | [iome.sh](https://iome.sh) and your I/O Mesh operator |
| Contributing | [CONTRIBUTING.md](CONTRIBUTING.md) |

## What we maintain

- Semantic versioning for the Go module (`vMAJOR.MINOR.PATCH`)  
- Security fixes on supported minor lines (see [SECURITY.md](SECURITY.md))  
- CI on every PR and `main`  

## What we do not provide in this repo

- Hosted broker uptime SLAs  
- Support for private monorepo / platform code paths  
- Guarantees about third-party Kafka or webhook partner systems  

## Before filing an issue

1. Run `go test ./...`  
2. Redact tokens, tenant IDs you consider sensitive, and customer payloads  
3. Include module version (`go list -m github.com/iome-sh/iomesh-client-sdk-go`) and Go version  
