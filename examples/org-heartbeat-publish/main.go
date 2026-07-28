// Command org-heartbeat-publish frames publish (+ optional pull) of organizational
// heartbeats (ops pulse) on dept.* streams via the public iomeshclient SDK.
//
// Public lexicon: heartbeat / pulse only. Offline stage smoke ≠ live APPLY.
// Beta / pre-1.0 surfaces — no invent GA. MIT edge client only.
//
// Env:
//
//	IOMESH_URL        mesh broker base (default http://127.0.0.1:8422)
//	IOMESH_TENANT     tenant (default dept.engineering)
//	IOMESH_ORG        optional X-IOMesh-Org
//	IOMESH_WORKSPACE  optional X-IOMesh-Workspace
//	IOMESH_API_KEY    optional Bearer
//	IOMESH_STREAM     stream name (default EVENTS)
//	IOMESH_SUBJECT    publish subject (default <tenant>.events.org-heartbeat)
//	IOMESH_PULL=1     also PullSubscribe + one Fetch (durable consumer demo)
//	IOMESH_CONSUMER   consumer name when IOMESH_PULL=1 (default sdk-org-heartbeat)
//
// Usage:
//
//	export IOMESH_URL=http://127.0.0.1:8422
//	go run ./examples/org-heartbeat-publish
//	# optional pull of the same org heartbeat subjects:
//	IOMESH_PULL=1 go run ./examples/org-heartbeat-publish
//
// For multi-cycle stage smoke (SUMMARY/RESULT scrapers), use examples/pull-loop.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
)

func main() {
	base := env("IOMESH_URL", "http://127.0.0.1:8422")
	tenant := env("IOMESH_TENANT", "dept.engineering")
	stream := env("IOMESH_STREAM", "EVENTS")
	subject := env("IOMESH_SUBJECT", tenant+".events.org-heartbeat")
	filter := tenant + ".events.>"

	opts := []iomeshclient.ConnectOpt{
		iomeshclient.WithTenant(tenant),
	}
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

	fmt.Printf("sdk=%s org-heartbeat framing (publish/pull dept.* pulse)\n", iomeshclient.Version)

	// Ensure a stream that accepts dept.* organizational heartbeats.
	info, err := nc.EnsureStream(ctx, iomeshclient.StreamConfig{
		Name:     stream,
		Subjects: []string{filter},
	})
	if err != nil {
		log.Fatalf("EnsureStream: %v", err)
	}
	if info != nil {
		fmt.Printf("PASS EnsureStream name=%s subjects=%v\n", info.Name, info.Subjects)
	} else {
		fmt.Printf("PASS EnsureStream name=%s (info nil after conflict is OK)\n", stream)
	}

	// Publish one organizational heartbeat (ops pulse). Agents consume org-tool events
	// on the same subject tree via PullSubscribe / ConsumerFetch.
	payload := []byte(`{"kind":"org_heartbeat","source":"iomesh-client-sdk-go","note":"examples/org-heartbeat-publish"}`)
	ack, err := nc.Publish(ctx, stream, subject, payload)
	if err != nil {
		log.Fatalf("Publish: %v", err)
	}
	fmt.Printf("PASS Publish org_heartbeat seq=%d subject=%s partition=%d\n", ack.Seq, ack.Subject, ack.Partition)

	// Optional: EmitDeptEvent is the structured metering / org-tool heartbeat helper
	// (POST /v1/streams/dept/publish). Warn-only so a local broker without dept stream
	// does not fail the example.
	if _, err := nc.EmitDeptEvent(ctx, iomeshclient.DeptEvent{
		Type:      "dept.agent.org_heartbeat",
		Tenant:    tenant,
		SessionID: tenant + ".org-heartbeat-example",
		Payload: map[string]any{
			"source": "iomesh-client-sdk-go",
			"probe":  "org-heartbeat-publish",
		},
	}); err != nil {
		log.Printf("WARN EmitDeptEvent: %v (dept stream may be absent offline)", err)
	} else {
		fmt.Println("PASS EmitDeptEvent type=dept.agent.org_heartbeat")
	}

	if os.Getenv("IOMESH_PULL") == "1" {
		consumer := env("IOMESH_CONSUMER", "sdk-org-heartbeat")
		sub, err := nc.PullSubscribe(ctx, iomeshclient.PullSubscribeConfig{
			Stream:   stream,
			Consumer: consumer,
			Filter:   filter,
		})
		if err != nil {
			log.Fatalf("PullSubscribe: %v", err)
		}
		fmt.Printf("PASS PullSubscribe stream=%s consumer=%s filter=%s\n", stream, consumer, filter)
		batch, err := sub.FetchContext(ctx, 5, iomeshclient.MaxWait(2*time.Second))
		if err != nil {
			log.Fatalf("FetchContext: %v", err)
		}
		fmt.Printf("PASS FetchContext count=%d\n", len(batch))
		if len(batch) > 0 {
			fmt.Print(iomeshclient.FormatMsgs(batch))
		}
	}

	fmt.Println("RESULT=done")
	fmt.Println("note: offline stage smoke ≠ live APPLY · dual_write default OFF · local-primary memory")
}

func env(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}
