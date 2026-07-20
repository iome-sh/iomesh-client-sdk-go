// Command pull-loop is a stage smoke for public SDK durable pull consumer
// (PullSubscribe → optional Publish → N× FetchContext → FormatMsgs → optional AckContext).
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
//	IOMESH_SUBJECT        optional filter_subject for the consumer (see resolveConsumerFilter)
//	IOMESH_BATCH          fetch batch size (default 5)
//	IOMESH_MAX_WAIT_MS    long-poll max wait ms (default 2000)
//	IOMESH_LOOPS          fetch cycle count (default 1; clamped to 1..100)
//	IOMESH_ENSURE_STREAM  set to 1 to EnsureStream with subject stream.>
//	IOMESH_PUBLISH        set to 1 to Publish one message before fetch
//	IOMESH_PUB_SUBJECT    publish subject override (see resolvePublishSubject priority)
//	IOMESH_ACK            set to 1 to AckContext fetched sequences each cycle
//	IOMESH_DELETE_CONSUMER set to 1 for best-effort DeleteConsumer after fetch loops
//
// Usage:
//
//	export IOMESH_URL=http://127.0.0.1:8422
//	export IOMESH_ENSURE_STREAM=1   # optional; defaults filter stream.> and pub under stream.>
//	export IOMESH_PUBLISH=1         # optional self-contained publish before fetch
//	export IOMESH_LOOPS=3           # optional multi-fetch cycles (default 1)
//	export IOMESH_ACK=1             # optional
//	export IOMESH_DELETE_CONSUMER=1 # optional cleanup after fetch loops
//	go run ./examples/pull-loop
//
// Always prints before RESULT=done:
//
//	SUMMARY cycles_completed=N fetch_total=M duration_ms=D
//
// Consumer filter defaults (resolveConsumerFilter):
//  1. IOMESH_SUBJECT if set (operator-chosen; used even if ensure is on)
//  2. When IOMESH_ENSURE_STREAM=1: stream.> (matches EnsureStream subjects)
//  3. Else empty (no filter / all subjects on stream)
//
// Publish subject defaults (when IOMESH_PUB_SUBJECT unset):
//  1. IOMESH_SUBJECT if set (operator-chosen; used even if ensure is on)
//  2. When IOMESH_ENSURE_STREAM=1: stream.sdk-pull-loop (matches EnsureStream subjects stream.>)
//  3. Else tenant+".sdk-pull-loop" if tenant set
//  4. Else stream+".demo"
//
// IOMESH_LOOPS fetch cycles then exit 0 (default one cycle). Errors after connect are warn-only.
// SUMMARY is always printed (cycles_completed / fetch_total / duration_ms wall clock).
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
	subjectEnv := strings.TrimSpace(os.Getenv("IOMESH_SUBJECT"))
	batch := envInt("IOMESH_BATCH", 5)
	maxWaitMS := envInt("IOMESH_MAX_WAIT_MS", 2000)
	if batch <= 0 {
		batch = 5
	}
	if maxWaitMS <= 0 {
		maxWaitMS = 2000
	}
	doPublish := os.Getenv("IOMESH_PUBLISH") == "1"
	ensureStream := os.Getenv("IOMESH_ENSURE_STREAM") == "1"
	doAck := os.Getenv("IOMESH_ACK") == "1"
	doDeleteConsumer := os.Getenv("IOMESH_DELETE_CONSUMER") == "1"
	loops := parseLoops(os.Getenv("IOMESH_LOOPS"), 1)
	// Resolve filter after subjectEnv so ensure-default stream.> is not passed as a publish subject.
	filter := resolveConsumerFilter(subjectEnv, ensureStream)
	pubSubject := publishSubject(subjectEnv, tenant, stream, ensureStream)

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

	// Wall clock for SUMMARY duration_ms (after connect opts resolved).
	start := time.Now()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: base}, opts...)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}

	// Budget: connect/status + optional ensure + create + optional pub + N long-poll fetches + optional ack.
	timeout := time.Duration(maxWaitMS)*time.Millisecond*time.Duration(loops) + 20*time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	fmt.Printf("sdk=%s user-agent=iomesh-client-sdk-go/%s\n", iomeshclient.Version, iomeshclient.Version)
	fmt.Printf("stream=%s consumer=%s batch=%d max_wait_ms=%d loops=%d filter=%q ensure_stream=%v publish=%v pub_subject=%q ack=%v delete_consumer=%v\n",
		stream, consumer, batch, maxWaitMS, loops, filter,
		ensureStream,
		doPublish,
		pubSubject,
		doAck,
		doDeleteConsumer,
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
	if ensureStream {
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
		printPullLoopDone(0, 0, start)
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

	// 3) Fetch cycles (FetchContext → FormatMsgs → optional AckContext); default one cycle.
	maxWait := iomeshclient.MaxWait(time.Duration(maxWaitMS) * time.Millisecond)
	cyclesCompleted := 0
	fetchTotal := 0
	for cycle := 1; cycle <= loops; cycle++ {
		msgs, err := sub.FetchContext(ctx, batch, maxWait)
		if err != nil {
			log.Printf("WARN FetchContext cycle=%d: %v", cycle, err)
			break
		}
		cyclesCompleted++
		fetchTotal += len(msgs)
		fmt.Print(iomeshclient.FormatMsgs(msgs))
		fmt.Printf("PASS FetchContext cycle=%d count=%d\n", cycle, len(msgs))

		if doAck && len(msgs) > 0 {
			seqs := make([]uint64, 0, len(msgs))
			for _, m := range msgs {
				if m != nil {
					seqs = append(seqs, m.Seq())
				}
			}
			if len(seqs) == 0 {
				log.Printf("WARN AckContext cycle=%d: no sequences", cycle)
			} else if err := sub.AckContext(ctx, seqs...); err != nil {
				log.Printf("WARN AckContext cycle=%d: %v", cycle, err)
			} else {
				fmt.Printf("PASS AckContext cycle=%d seqs=%v\n", cycle, seqs)
			}
		}
	}

	// 4) Optional best-effort DeleteConsumer after fetch loops (warn-only on fail)
	if doDeleteConsumer {
		if err := nc.DeleteConsumer(ctx, stream, consumer); err != nil {
			log.Printf("WARN DeleteConsumer stream=%s consumer=%s: %v", stream, consumer, err)
		} else {
			fmt.Printf("PASS DeleteConsumer stream=%s consumer=%s\n", stream, consumer)
		}
	}

	printPullLoopDone(cyclesCompleted, fetchTotal, start)
}

// printPullLoopDone emits SUMMARY then RESULT=done using wall-clock duration since start.
func printPullLoopDone(cyclesCompleted, fetchTotal int, start time.Time) {
	durationMS := int(time.Since(start).Milliseconds())
	fmt.Println(formatPullLoopSummary(cyclesCompleted, fetchTotal, durationMS))
	fmt.Println("RESULT=done")
}

// formatPullLoopSummary returns the stage-smoke SUMMARY line for pull-loop.
// durationMS is clamped to >= 0. Pure helper (no I/O).
func formatPullLoopSummary(cyclesCompleted, fetchTotal, durationMS int) string {
	if durationMS < 0 {
		durationMS = 0
	}
	return fmt.Sprintf("SUMMARY cycles_completed=%d fetch_total=%d duration_ms=%d",
		cyclesCompleted, fetchTotal, durationMS)
}

// parseLoops returns fetch cycle count from an env value.
// Empty or invalid → def (then clamped). Result is always in [1, 100].
func parseLoops(env string, def int) int {
	n := def
	if v := strings.TrimSpace(env); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			n = parsed
		}
	}
	if n < 1 {
		return 1
	}
	if n > 100 {
		return 100
	}
	return n
}

// publishSubject resolves the publish subject from env and flags.
// subjectEnv is the raw IOMESH_SUBJECT (not the resolved consumer filter) so an ensure-default
// filter of stream.> does not become the publish subject.
// See resolvePublishSubject for priority.
func publishSubject(subjectEnv, tenant, stream string, ensureStream bool) string {
	return resolvePublishSubject(os.Getenv("IOMESH_PUB_SUBJECT"), subjectEnv, tenant, stream, ensureStream)
}

// resolveConsumerFilter picks a durable pull consumer filter_subject:
//  1. subjectEnv (IOMESH_SUBJECT) if set — operator-chosen even when ensure is on
//  2. when ensureStream: "stream.>" (matches EnsureStream subjects)
//  3. else empty (no filter / all subjects on stream)
func resolveConsumerFilter(subjectEnv string, ensureStream bool) string {
	if s := strings.TrimSpace(subjectEnv); s != "" {
		return s
	}
	if ensureStream {
		return "stream.>"
	}
	return ""
}

// resolvePublishSubject picks a publish subject in priority order:
//  1. pubSubject (IOMESH_PUB_SUBJECT) if set
//  2. filter (IOMESH_SUBJECT) if set — operator-chosen even when ensure is on
//  3. when ensureStream: "stream.sdk-pull-loop" (under EnsureStream subjects stream.>)
//  4. tenant+".sdk-pull-loop" if tenant set
//  5. stream+".demo"
//
// filter is the raw IOMESH_SUBJECT, not resolveConsumerFilter's ensure default (stream.>).
func resolvePublishSubject(pubSubject, filter, tenant, stream string, ensureStream bool) string {
	if s := strings.TrimSpace(pubSubject); s != "" {
		return s
	}
	if filter != "" {
		return filter
	}
	if ensureStream {
		return "stream.sdk-pull-loop"
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
