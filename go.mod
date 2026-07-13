module github.com/iome-sh/iomesh-client-sdk-go

// Public SDK: support last two stable Go lines in CI (see .github/workflows/ci.yml).
go 1.23.0

require github.com/nrednav/cuid2 v1.1.0

require (
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
)
