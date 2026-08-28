// Command github-stream-read replays messages from a GitHub-ingested stream
// the caller is allowed to see (ListStreamMessages).
//
// This is stream replay, not an org-health or heart-rate API.
// Slack and PagerDuty are not live pulses in this example.
// Chat is not the record. Empty list is honest (no signed GitHub event yet).
// Offline httptest ≠ live APPLY. Surfaces are Beta / pre-1.0.
//
// Env:
//
//	IOMESH_URL        mesh broker base (default http://127.0.0.1:8422)
//	IOMESH_TENANT     tenant (default dept.engineering)
//	IOMESH_ORG        optional X-IOMesh-Org
//	IOMESH_WORKSPACE  optional X-IOMesh-Workspace
//	IOMESH_API_KEY    optional Bearer
//	IOMESH_STREAM     stream name (default OPERATIONAL_EVENTS)
//	IOMESH_LIMIT      replay limit (default 50, cap 1000)
//
// Usage:
//
//	export IOMESH_URL=http://127.0.0.1:8422
//	go run ./examples/github-stream-read
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
)

func main() {
	base := env("IOMESH_URL", "http://127.0.0.1:8422")
	tenant := env("IOMESH_TENANT", "dept.engineering")
	stream := env("IOMESH_STREAM", "OPERATIONAL_EVENTS")
	limit := envInt("IOMESH_LIMIT", 50)

	opts := []iomeshclient.ConnectOpt{iomeshclient.WithTenant(tenant)}
	if org := strings.TrimSpace(os.Getenv("IOMESH_ORG")); org != "" {
		opts = append(opts, iomeshclient.WithOrg(org))
	}
	if ws := strings.TrimSpace(os.Getenv("IOMESH_WORKSPACE")); ws != "" {
		opts = append(opts, iomeshclient.WithWorkspace(ws))
	}
	if key := strings.TrimSpace(os.Getenv("IOMESH_API_KEY")); key != "" {
		opts = append(opts, iomeshclient.WithBearerToken(key))
	}

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: base}, opts...)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("sdk=%s github-stream-read stream=%s (ListStreamMessages · not org-health)\n", iomeshclient.Version, stream)

	msgs, err := nc.ListStreamMessages(ctx, stream, iomeshclient.ListStreamMessagesOptions{Limit: limit})
	if err != nil {
		log.Fatalf("ListStreamMessages: %v", err)
	}
	fmt.Print(formatGitHubReplay(stream, msgs))
}

func formatGitHubReplay(stream string, msgs []iomeshclient.StreamMessage) string {
	var b strings.Builder
	fmt.Fprintf(&b, "replay stream=%s count=%d · GitHub-ingested events the caller may see\n", stream, len(msgs))
	fmt.Fprintf(&b, "honesty: not org-health · not heart-rate · Slack/PagerDuty not pulses here · chat is not the record\n")
	if len(msgs) == 0 {
		fmt.Fprintf(&b, "(empty) no signed GitHub event on this stream yet · catalog ≠ Connected\n")
		return b.String()
	}
	for _, m := range msgs {
		fmt.Fprintf(&b, "seq=%d subject=%s bytes=%d\n", m.Seq, m.Subject, len(m.Payload))
	}
	return b.String()
}

func env(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
