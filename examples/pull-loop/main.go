// Command pull-loop is a stage smoke for public SDK durable pull consumer
// (PullSubscribe → optional Publish → FetchContext → FormatMsgs → optional AckContext).
//
// Env:
//
//	IOMESH_URL            mesh broker base (required)
//	IOMESH_TENANT         tenant (default demo.tenant)
//	IOMESH_ORG            optional X-IOMesh-Org
//	IOMESH_WORKSPACE      optional X-IOMesh-Workspace
//	IOMESH_API_KEY        optional Bearer
//	IOMESH_STREAM         stream name (default EVENTS)
//	IOMESH_CONSUMER       durable consumer name (default sdk-pull-loop)
//	IOMESH_SUBJECT        optional filter_subject for the consumer
//	IOMESH_BATCH          fetch batch size (default 5)
//	IOMESH_MAX_WAIT_MS    long-poll max wait ms (default 2000)
//	IOMESH_ENSURE_STREAM  set to 1 to EnsureStream with subject stream.>
//	IOMESH_PUBLISH        set to 1 to Publish one message before fetch
//	IOMESH_PUB_SUBJECT    publish subject (default: filter, else tenant+".sdk-pull-loop", else stream+".demo")
//	IOMESH_ACK            set to 1 to AckContext fetched sequences
//
// Usage:
//
//	export IOMESH_URL=http://127.0.0.1:8422
//	export IOMESH_ENSURE_STREAM=1   # optional
//	export IOMESH_PUBLISH=1         # optional self-contained publish before fetch
//	export IOMESH_ACK=1             # optional
//	go run ./examples/pull-loop
//
// When IOMESH_PUBLISH=1 with EnsureStream (stream.>), set IOMESH_PUB_SUBJECT under that
// prefix (e.g. stream.demo) so the message is accepted and fetchable.
//
// One fetch cycle then exit 0. Errors after connect are warn-only.
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
	base := env("IOMESH_URL", "")
	if base == "" {
		log.Fatal("IOMESH_URL required")
	}
	tenant := env("IOMESH_TENANT", "demo.tenant")
	stream := env("IOMESH_STREAM", "EVENTS")
	consumer := env("IOMESH_CONSUMER", "sdk-pull-loop")
	filter := strings.TrimSpace(os.Getenv("IOMESH_SUBJECT"))
	batch := envInt("IOMESH_BATCH", 5)
	maxWaitMS := envInt("IOMESH_MAX_WAIT_MS", 2000)
	if batch <= 0 {
		batch = 5
	}
	if maxWaitMS <= 0 {
		maxWaitMS = 2000
	}
	doPublish := os.Getenv("IOMESH_PUBLISH") == "1"
	pubSubject := publishSubject(filter, tenant, stream)

	opts := []iomeshclient.ConnectOpt{
		iomeshclient.WithTenant(tenant),
	}
	if org := os.Getenv("IOMESH_ORG"); org != "" {
		opts = append(opts, iomeshclient.WithOrg(org))
	}
	if ws := os.Getenv("IOMESH_WORKSPACE"); ws != "" {
		opts = append(opts, iomeshclient.WithWorkspace(ws))
	}
	if key := os.Getenv("IOMESH_API_KEY"); key != "" {
		opts = append(opts, iomeshclient.WithBearerToken(key))
	}

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: base}, opts...)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}

	// Budget: connect/status + optional ensure + create + optional pub + one long-poll fetch + optional ack.
	timeout := time.Duration(maxWaitMS)*time.Millisecond + 20*time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	fmt.Printf("sdk=%s user-agent=iomesh-client-sdk-go/%s\n", iomeshclient.Version, iomeshclient.Version)
	fmt.Printf("stream=%s consumer=%s batch=%d max_wait_ms=%d filter=%q ensure_stream=%v publish=%v pub_subject=%q ack=%v\n",
		stream, consumer, batch, maxWaitMS, filter,
		os.Getenv("IOMESH_ENSURE_STREAM") == "1",
		doPublish,
		pubSubject,
		os.Getenv("IOMESH_ACK") == "1",
	)

	// 0) ConnectionStatus snapshot (identity + Health + Ready; fail-open)
	st := nc.ConnectionStatus(ctx)
	fmt.Print(iomeshclient.FormatConnectionStatus(st))
	if !st.HealthOK {
		log.Printf("WARN Health: %s", st.HealthErr)
	} else {
		fmt.Println("PASS Health GET /health")
	}
	if !st.ReadyOK {
		log.Printf("WARN Ready: %s", st.ReadyErr)
	} else {
		fmt.Println("PASS Ready")
	}

	// 1) Optional EnsureStream (subject stream.>)
	if os.Getenv("IOMESH_ENSURE_STREAM") == "1" {
		info, err := nc.EnsureStream(ctx, iomeshclient.StreamConfig{
			Name:     stream,
			Subjects: []string{"stream.>"},
		})
		if err != nil {
			log.Printf("WARN EnsureStream stream=%s: %v", stream, err)
		} else {
			fmt.Printf("PASS EnsureStream stream=%s", stream)
			if info != nil {
				fmt.Printf(" subjects=%v messages=%d", info.Subjects, info.Messages)
			}
			fmt.Println()
		}
	}

	// 2) PullSubscribe (CreateConsumer + subscription handle)
	sub, err := nc.PullSubscribe(ctx, iomeshclient.PullSubscribeConfig{
		Stream:   stream,
		Consumer: consumer,
		Filter:   filter,
	})
	if err != nil {
		log.Printf("WARN PullSubscribe stream=%s consumer=%s: %v", stream, consumer, err)
		fmt.Println("RESULT=done")
		return
	}
	fmt.Print(iomeshclient.FormatConsumerInfo(sub.ConsumerInfo()))
	fmt.Printf("PASS PullSubscribe stream=%s consumer=%s\n", stream, consumer)

	// 2b) Optional self-contained Publish before fetch (warn-only on fail)
	if doPublish {
		payload := []byte(fmt.Sprintf(`{"source":"sdk-pull-loop","ts":%d}`, time.Now().Unix()))
		ack, err := nc.Publish(ctx, stream, pubSubject, payload)
		if err != nil {
			log.Printf("WARN Publish stream=%s subject=%s: %v", stream, pubSubject, err)
		} else {
			fmt.Printf("PASS Publish stream=%s subject=%s", stream, pubSubject)
			if ack != nil {
				fmt.Printf(" seq=%d", ack.Seq)
			}
			fmt.Println()
		}
	}

	// 3) One fetch cycle (FetchContext → FormatMsgs → optional AckContext)
	msgs, err := sub.FetchContext(ctx, batch, iomeshclient.MaxWait(time.Duration(maxWaitMS)*time.Millisecond))
	if err != nil {
		log.Printf("WARN FetchContext: %v", err)
		fmt.Println("RESULT=done")
		return
	}
	fmt.Print(iomeshclient.FormatMsgs(msgs))
	fmt.Printf("PASS FetchContext count=%d\n", len(msgs))

	if os.Getenv("IOMESH_ACK") == "1" && len(msgs) > 0 {
		seqs := make([]uint64, 0, len(msgs))
		for _, m := range msgs {
			if m != nil {
				seqs = append(seqs, m.Seq())
			}
		}
		if len(seqs) == 0 {
			log.Printf("WARN AckContext: no sequences")
		} else if err := sub.AckContext(ctx, seqs...); err != nil {
			log.Printf("WARN AckContext: %v", err)
		} else {
			fmt.Printf("PASS AckContext seqs=%v\n", seqs)
		}
	}

	fmt.Println("RESULT=done")
}

// publishSubject resolves IOMESH_PUB_SUBJECT, else filter, else tenant+".sdk-pull-loop", else stream+".demo".
func publishSubject(filter, tenant, stream string) string {
	if s := strings.TrimSpace(os.Getenv("IOMESH_PUB_SUBJECT")); s != "" {
		return s
	}
	if filter != "" {
		return filter
	}
	if tenant != "" {
		return tenant + ".sdk-pull-loop"
	}
	return stream + ".demo"
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
