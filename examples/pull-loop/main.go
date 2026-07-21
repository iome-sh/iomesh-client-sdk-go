// Command pull-loop is a stage smoke for public SDK durable pull consumer
// (PullSubscribe → optional pre-loop Publish → N× (optional per-cycle Publish →
// FetchContext → FormatMsgs → optional AckContext)).
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
//	IOMESH_PUBLISH        set to 1 to Publish one message before the fetch loop
//	IOMESH_PUBLISH_EACH   set to 1 to Publish one message at the start of each cycle
//	IOMESH_PUB_SUBJECT    publish subject override (see resolvePublishSubject priority)
//	IOMESH_ACK            set to 1 to AckContext fetched sequences each cycle
//	IOMESH_DELETE_CONSUMER set to 1 for best-effort sub.Delete after fetch loops
//	IOMESH_WAIT_READY_MS  optional WaitReady preflight budget ms after ConnectionStatus
//	                      (0/unset = skip; invalid → 0; clamped to max 120000)
//	IOMESH_WAIT_INTERVAL_MS optional WaitReady poll interval ms (default 500; only when WAIT_READY_MS>0)
//	IOMESH_WAIT_REQUIRE_HEALTH set to 1 so WaitReady preflight also requires Health
//	                      (only applies when IOMESH_WAIT_READY_MS>0; default false)
//	IOMESH_STRICT         set to 1 for non-zero exit (1) on stage smoke hard failures
//	                      (ConnectionStatus probe aggregate uses result=err; see STRICT note)
//
// Publish semantics:
//
//	IOMESH_PUBLISH=1 only          → one publish before the fetch loop (current default)
//	IOMESH_PUBLISH_EACH=1          → publish at the start of each cycle (including first)
//	both set                       → EACH wins for cycles; pre-loop single publish is skipped
//	                                 so the first cycle is not double-published
//
// Usage:
//
//	export IOMESH_URL=http://127.0.0.1:8422
//	export IOMESH_ENSURE_STREAM=1   # optional; defaults filter stream.> and pub under stream.>
//	export IOMESH_PUBLISH=1         # optional one-shot publish before the fetch loop
//	export IOMESH_PUBLISH_EACH=1    # optional publish at start of each cycle (self-contained multi-fetch)
//	export IOMESH_LOOPS=3           # optional multi-fetch cycles (default 1)
//	export IOMESH_ACK=1             # optional
//	export IOMESH_DELETE_CONSUMER=1 # optional cleanup after fetch loops
//	export IOMESH_WAIT_READY_MS=5000 # optional WaitReady preflight budget (ms)
//	export IOMESH_WAIT_INTERVAL_MS=250 # optional WaitReady poll interval (ms; default 500)
//	export IOMESH_WAIT_REQUIRE_HEALTH=1 # optional; WaitReady also requires Health
//	export IOMESH_STRICT=1          # optional; exit 1 after SUMMARY on hard stage failures
//	go run ./examples/pull-loop
//
// Always prints before RESULT=done:
//
//	SUMMARY version=V user_agent=UA base_url=B tenant=T org=O workspace=W cycles_completed=N fetch_total=M duration_ms=D wait_ready_ms=W wait_interval_ms=I wait_require_health=B wait_ready_attempts=A failed=F strict=S result=R exit_code=E
//	RESULT=done version=V user_agent=UA base_url=B tenant=T org=O workspace=W result=R exit_code=E
//
// version is the SDK package Version (leading on SUMMARY; also on RESULT so
// scrapers can key it without re-parsing the banner sdk= line). user_agent is always
// emitted after version= before base_url= on both SUMMARY and RESULT (package default
// User-Agent iomesh-client-sdk-go/<Version>, same string ConnectionStatus uses when
// WithUserAgent is unset; empty string still emits user_agent= if truly unset) so
// scrapers peer ConnectionStatus user_agent without inventing readiness. base_url is
// always emitted after user_agent= before tenant= (connect mesh URL / IOMESH_URL,
// same string ConnectionStatus uses as base_url; empty string still emits base_url=
// if truly unset) so scrapers peer ConnectionStatus base_url without inventing
// readiness. SUMMARY and RESULT always emit tenant/org/workspace identity from
// connect env (IOMESH_TENANT default demo.tenant; IOMESH_ORG / IOMESH_WORKSPACE empty
// string honest when unset) so scrapers peer ConnectionStatus identity without
// inventing readiness. SUMMARY and RESULT always emit result=ok|err derived from the
// existing failed flag (ok when failed==false; err when failed==true), peers
// ConnectionStatus.Result / AggregateConnectionResult without inventing readiness.
// RESULT keeps the RESULT=done token and always emits version, user_agent, base_url,
// identity, result, and exit_code (exit_code semantics match SUMMARY: 1 only when
// strict && failed; otherwise 0).
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
// Default (IOMESH_STRICT unset): warn-only after connect + exit 0 with RESULT=done / SUMMARY.
// IOMESH_STRICT=1: still prints SUMMARY when possible, then exit 1 if any hard stage failure:
// ConnectionStatus.result=err (Health/Ready probe aggregate; per-probe PASS/WARN still printed),
// WaitReady error (when IOMESH_WAIT_READY_MS>0), EnsureStream error, PullSubscribe error,
// Publish error (when IOMESH_PUBLISH / IOMESH_PUBLISH_EACH requested), FetchContext error,
// or DeleteConsumer/sub.Delete error (when IOMESH_DELETE_CONSUMER=1).
// Connect failure already uses log.Fatal (exit non-zero).
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
	// org/workspace are optional connect headers; empty string honest when unset
	// (always emitted on SUMMARY identity knobs regardless of whether opts were set).
	org := os.Getenv("IOMESH_ORG")
	workspace := os.Getenv("IOMESH_WORKSPACE")
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
	publishEach := wantPublishEach(os.Getenv("IOMESH_PUBLISH_EACH"))
	ensureStream := os.Getenv("IOMESH_ENSURE_STREAM") == "1"
	doAck := os.Getenv("IOMESH_ACK") == "1"
	doDeleteConsumer := os.Getenv("IOMESH_DELETE_CONSUMER") == "1"
	strict := envStrict(os.Getenv("IOMESH_STRICT"))
	waitReadyMS := parseWaitReadyMS(os.Getenv("IOMESH_WAIT_READY_MS"))
	waitIntervalMS := parseWaitIntervalMS(os.Getenv("IOMESH_WAIT_INTERVAL_MS"))
	waitRequireHealth := envWaitRequireHealth(os.Getenv("IOMESH_WAIT_REQUIRE_HEALTH"))
	loops := parseLoops(os.Getenv("IOMESH_LOOPS"), 1)
	// Resolve filter after subjectEnv so ensure-default stream.> is not passed as a publish subject.
	filter := resolveConsumerFilter(subjectEnv, ensureStream)
	pubSubject := publishSubject(subjectEnv, tenant, stream, ensureStream)

	opts := []iomeshclient.ConnectOpt{
		iomeshclient.WithTenant(tenant),
	}
	if org != "" {
		opts = append(opts, iomeshclient.WithOrg(org))
	}
	if workspace != "" {
		opts = append(opts, iomeshclient.WithWorkspace(workspace))
	}
	if key := os.Getenv("IOMESH_API_KEY"); key != "" {
		opts = append(opts, iomeshclient.WithBearerToken(key))
	}

	// Wall clock for SUMMARY duration_ms (after connect opts resolved).
	start := time.Now()
	failed := false
	// waitReadyAttempts is set when WaitReady preflight runs; stays 0 when wait off.
	waitReadyAttempts := 0

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: base}, opts...)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}

	// Budget: connect/status + optional WaitReady + ensure + create + optional pub + N long-poll fetches + optional ack.
	timeout := time.Duration(maxWaitMS)*time.Millisecond*time.Duration(loops) + 20*time.Second
	if waitReadyMS > 0 {
		timeout += time.Duration(waitReadyMS) * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	fmt.Printf("sdk=%s user-agent=iomesh-client-sdk-go/%s\n", iomeshclient.Version, iomeshclient.Version)
	fmt.Printf("stream=%s consumer=%s batch=%d max_wait_ms=%d loops=%d filter=%q ensure_stream=%v publish=%v publish_each=%v pub_subject=%q ack=%v delete_consumer=%v wait_ready_ms=%d wait_interval_ms=%d wait_require_health=%v strict=%v\n",
		stream, consumer, batch, maxWaitMS, loops, filter,
		ensureStream,
		doPublish,
		publishEach,
		pubSubject,
		doAck,
		doDeleteConsumer,
		waitReadyMS,
		waitIntervalMS,
		waitRequireHealth,
		strict,
	)

	// 0) ConnectionStatus snapshot (identity + Health + Ready; fail-open unless IOMESH_STRICT=1).
	// STRICT probe fail uses ConnectionStatus.result aggregate once (covers Health + Ready).
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
	if statusResultFailed(st.Result) {
		failed = true
		log.Printf("WARN ConnectionStatus result=err")
	}

	// 0b) Optional WaitReady preflight (after status, before EnsureStream).
	// When IOMESH_WAIT_READY_MS>0: poll Ready with budget ms; interval from
	// IOMESH_WAIT_INTERVAL_MS (default 500ms). IOMESH_WAIT_REQUIRE_HEALTH=1 sets
	// RequireHealth (only applies when budget > 0).
	// Failure is warn-only by default; under IOMESH_STRICT=1 sets failed.
	if waitReadyMS > 0 {
		wrCtx, wrCancel := context.WithTimeout(ctx, time.Duration(waitReadyMS)*time.Millisecond)
		elapsed, attempts, wrErr := nc.WaitReadyAttempts(wrCtx, iomeshclient.WaitReadyOptions{
			Interval:      time.Duration(waitIntervalMS) * time.Millisecond,
			RequireHealth: waitRequireHealth,
		})
		wrCancel()
		elapsedMS := int(elapsed.Milliseconds())
		if elapsedMS < 0 {
			elapsedMS = 0
		}
		if attempts < 0 {
			attempts = 0
		}
		waitReadyAttempts = attempts
		if wrErr != nil {
			log.Printf("WARN WaitReady: %v elapsed_ms=%d interval_ms=%d require_health=%v attempts=%d", wrErr, elapsedMS, waitIntervalMS, waitRequireHealth, attempts)
			failed = true
		} else {
			fmt.Printf("PASS WaitReady elapsed_ms=%d interval_ms=%d require_health=%v attempts=%d\n", elapsedMS, waitIntervalMS, waitRequireHealth, attempts)
		}
	}

	// 1) Optional EnsureStream (subject stream.>)
	if ensureStream {
		info, err := nc.EnsureStream(ctx, iomeshclient.StreamConfig{
			Name:     stream,
			Subjects: []string{"stream.>"},
		})
		if err != nil {
			log.Printf("WARN EnsureStream stream=%s: %v", stream, err)
			failed = true
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
		failed = true
		finishPullLoop(0, 0, start, waitReadyMS, waitIntervalMS, waitRequireHealth, waitReadyAttempts, strict, failed, base, tenant, org, workspace)
		return
	}
	fmt.Print(iomeshclient.FormatSubscription(sub))
	fmt.Printf("PASS PullSubscribe stream=%s consumer=%s\n", stream, consumer)

	// 2b) Optional one-shot Publish before the fetch loop (warn-only on fail unless STRICT).
	// Skipped when IOMESH_PUBLISH_EACH=1 so the first cycle is not double-published.
	if doPublish && !publishEach {
		if !publishDemo(ctx, nc, stream, pubSubject) {
			failed = true
		}
	}

	// 3) Fetch cycles (optional per-cycle Publish → FetchContext → FormatMsgs → optional AckContext).
	maxWait := iomeshclient.MaxWait(time.Duration(maxWaitMS) * time.Millisecond)
	cyclesCompleted := 0
	fetchTotal := 0
	for cycle := 1; cycle <= loops; cycle++ {
		// Per-cycle publish before fetch (warn-only on fail unless STRICT; still fetch).
		if publishEach {
			if !publishDemo(ctx, nc, stream, pubSubject) {
				failed = true
			}
		}

		msgs, err := sub.FetchContext(ctx, batch, maxWait)
		if err != nil {
			log.Printf("WARN FetchContext cycle=%d: %v", cycle, err)
			failed = true
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

	// 4) Optional best-effort sub.Delete after fetch loops (warn-only on fail unless STRICT)
	if doDeleteConsumer {
		if err := sub.Delete(ctx); err != nil {
			log.Printf("WARN Delete stream=%s consumer=%s: %v", stream, consumer, err)
			failed = true
		} else {
			fmt.Printf("PASS Delete stream=%s consumer=%s\n", stream, consumer)
		}
	}

	finishPullLoop(cyclesCompleted, fetchTotal, start, waitReadyMS, waitIntervalMS, waitRequireHealth, waitReadyAttempts, strict, failed, base, tenant, org, workspace)
}

// publishDemo publishes one self-contained demo payload.
// Returns false on Publish error (caller may mark failed under IOMESH_STRICT).
func publishDemo(ctx context.Context, nc *iomeshclient.Client, stream, pubSubject string) bool {
	payload := []byte(fmt.Sprintf(`{"source":"sdk-pull-loop","ts":%d}`, time.Now().Unix()))
	ack, err := nc.Publish(ctx, stream, pubSubject, payload)
	if err != nil {
		log.Printf("WARN Publish stream=%s subject=%s: %v", stream, pubSubject, err)
		return false
	}
	fmt.Printf("PASS Publish stream=%s subject=%s", stream, pubSubject)
	if ack != nil {
		fmt.Printf(" seq=%d", ack.Seq)
	}
	fmt.Println()
	return true
}

// finishPullLoop emits SUMMARY then RESULT=done version=V user_agent=UA base_url=B tenant=T org=O workspace=W result=R exit_code=E;
// under IOMESH_STRICT exits 1 when failed.
// version (SDK package Version), user_agent, base_url (connect mesh URL), identity
// tenant/org/workspace, WaitReady knobs (including wait_ready_attempts), failed, strict,
// result (ok|err from failed), and exit_code are always included on SUMMARY (0/false when
// WaitReady off; empty strings honest when unset). version, user_agent, base_url, identity,
// result, and exit_code are always on RESULT too; result peers ConnectionStatus.Result
// (ok when !failed; err when failed); exit_code matches the process exit used after
// SUMMARY (1 only when strict && failed; otherwise 0, including non-strict failures).
func finishPullLoop(cyclesCompleted, fetchTotal int, start time.Time, waitReadyMS, waitIntervalMS int, waitRequireHealth bool, waitReadyAttempts int, strict, failed bool, baseURL, tenant, org, workspace string) {
	printPullLoopDone(cyclesCompleted, fetchTotal, start, waitReadyMS, waitIntervalMS, waitRequireHealth, waitReadyAttempts, failed, strict, baseURL, tenant, org, workspace)
	if strict && failed {
		os.Exit(1)
	}
}

// printPullLoopDone emits SUMMARY then RESULT=done version=V user_agent=UA base_url=B tenant=T org=O workspace=W result=R exit_code=E
// using wall-clock duration since start.
// version is always iomeshclient.Version (leading field on SUMMARY; also on RESULT).
// user_agent is always package default "iomesh-client-sdk-go/<Version>" (same string
// ConnectionStatus uses when WithUserAgent is unset; after version= before base_url=).
// base_url is always the connect mesh URL / IOMESH_URL (same string ConnectionStatus uses
// as base_url; after user_agent= before tenant=; empty honest when unset).
// tenant/org/workspace are always on SUMMARY and RESULT (connect identity; empty honest when unset).
// result is always ok|err from the failed flag (peers ConnectionStatus.Result).
// exit_code on RESULT matches SUMMARY / process exit (1 only when strict && failed).
func printPullLoopDone(cyclesCompleted, fetchTotal int, start time.Time, waitReadyMS, waitIntervalMS int, waitRequireHealth bool, waitReadyAttempts int, failed, strict bool, baseURL, tenant, org, workspace string) {
	durationMS := int(time.Since(start).Milliseconds())
	// Package default User-Agent (mirrors unexported defaultUserAgent / ConnectionStatus).
	userAgent := "iomesh-client-sdk-go/" + iomeshclient.Version
	fmt.Println(formatPullLoopSummary(cyclesCompleted, fetchTotal, durationMS, waitReadyMS, waitIntervalMS, waitRequireHealth, waitReadyAttempts, failed, strict, iomeshclient.Version, userAgent, baseURL, tenant, org, workspace))
	exitCode := 0
	if strict && failed {
		exitCode = 1
	}
	fmt.Println(formatPullLoopResult(exitCode, failed, iomeshclient.Version, userAgent, baseURL, tenant, org, workspace))
}

// envStrict reports whether IOMESH_STRICT enables hard-fail exit after SUMMARY.
// Only the exact value "1" is truthy (matches other pull-loop flag env convention).
func envStrict(v string) bool {
	return v == "1"
}

// envWaitRequireHealth reports whether IOMESH_WAIT_REQUIRE_HEALTH enables
// RequireHealth on the optional WaitReady preflight. True only when trimmed == "1".
// Only applies when IOMESH_WAIT_READY_MS > 0 (default false).
func envWaitRequireHealth(v string) bool {
	return strings.TrimSpace(v) == "1"
}

// statusResultFailed reports whether ConnectionStatus.Result is the aggregate fail
// signal "err". Used under IOMESH_STRICT so Health/Ready probe failures mark failed
// once via result (per-probe PASS/WARN lines stay independent).
func statusResultFailed(result string) bool {
	return result == "err"
}

// wantPublishEach reports whether IOMESH_PUBLISH_EACH enables per-cycle publish.
// Only the exact value "1" is truthy (matches other pull-loop flag env convention).
func wantPublishEach(env string) bool {
	return env == "1"
}

// formatPullLoopSummary returns the stage-smoke SUMMARY line for pull-loop.
// Field order (peers ConnectionStatus field family for scrapers):
//
//	version= user_agent= base_url= tenant= org= workspace= …
//
// version is always emitted first (caller passes iomeshclient.Version) so scrapers
// can key SDK version without re-parsing the banner sdk= line; empty string still
// emits version= (field present).
// user_agent is always emitted after version= before base_url= (caller passes package
// default "iomesh-client-sdk-go/<Version>" matching ConnectionStatus / defaultUserAgent
// when WithUserAgent is unset; empty string still emits user_agent= when truly unset)
// so scrapers peer ConnectionStatus user_agent without inventing readiness.
// base_url is always emitted after user_agent= before tenant= (caller passes connect
// mesh URL / IOMESH_URL matching ConnectionStatus.BaseURL; empty string still emits
// base_url= when truly unset) so scrapers peer ConnectionStatus base_url without
// inventing readiness.
// tenant/org/workspace are always emitted next (connect identity from IOMESH_TENANT
// / IOMESH_ORG / IOMESH_WORKSPACE; empty string still emits tenant= org= workspace=
// when unset) so scrapers peer ConnectionStatus identity without inventing readiness.
// durationMS is clamped to >= 0. WaitReady knobs are always emitted:
// wait_ready_ms is the configured budget (0 when off); when wait is off,
// wait_interval_ms is 0, wait_require_health is false, and wait_ready_attempts
// is 0 so scrapers see "not used"; when wait is on, wait_interval_ms is the
// effective poll interval (parseWaitIntervalMS), wait_require_health is the
// RequireHealth flag, and wait_ready_attempts is the probe cycle count from
// WaitReadyAttempts.
// failed is always emitted (same boolean used for IOMESH_STRICT exit) so
// scrapers see hard stage-smoke failures even when non-strict (exit 0).
// strict is always emitted (IOMESH_STRICT mode) so scrapers see whether
// hard-fail exit was enabled without re-parsing env/banner.
// result is always emitted after strict= before exit_code= as ok|err derived
// from failed (pullLoopResult; peers ConnectionStatus.Result).
// exit_code is always emitted and matches process exit after SUMMARY:
// 1 only when strict && failed; otherwise 0 (including non-strict with failed).
// Pure helper (no I/O); version, user_agent, base_url, and identity are parameters for testability.
func formatPullLoopSummary(cyclesCompleted, fetchTotal, durationMS, waitReadyMS, waitIntervalMS int, waitRequireHealth bool, waitReadyAttempts int, failed, strict bool, version, userAgent, baseURL, tenant, org, workspace string) string {
	if durationMS < 0 {
		durationMS = 0
	}
	if waitReadyMS <= 0 {
		waitReadyMS = 0
		waitIntervalMS = 0
		waitRequireHealth = false
		waitReadyAttempts = 0
	}
	exitCode := 0
	if strict && failed {
		exitCode = 1
	}
	return fmt.Sprintf("SUMMARY version=%s user_agent=%s base_url=%s tenant=%s org=%s workspace=%s cycles_completed=%d fetch_total=%d duration_ms=%d wait_ready_ms=%d wait_interval_ms=%d wait_require_health=%v wait_ready_attempts=%d failed=%t strict=%t result=%s exit_code=%d",
		version, userAgent, baseURL, tenant, org, workspace, cyclesCompleted, fetchTotal, durationMS, waitReadyMS, waitIntervalMS, waitRequireHealth, waitReadyAttempts, failed, strict, pullLoopResult(failed), exitCode)
}

// pullLoopResult maps the stage-smoke failed flag to ConnectionStatus-style result.
// Peers AggregateConnectionResult / ConnectionStatus.Result: "ok" when no hard
// failure, "err" when failed. Pure helper (no I/O).
func pullLoopResult(failed bool) string {
	if failed {
		return "err"
	}
	return "ok"
}

// formatPullLoopResult returns the stage-smoke RESULT line for pull-loop.
// Field order (matches SUMMARY / peers ConnectionStatus field family):
//
//	RESULT=done version= user_agent= base_url= tenant= org= workspace= result= exit_code=
//
// Keeps the RESULT=done token for scrapers that key off it, and always emits
// version (caller passes iomeshclient.Version; empty string still emits version=),
// user_agent after version= before base_url= (same package default as SUMMARY /
// ConnectionStatus; empty string still emits user_agent= when truly unset),
// base_url after user_agent= before tenant= (same connect mesh URL as SUMMARY /
// ConnectionStatus.BaseURL; empty string still emits base_url= when truly unset),
// tenant/org/workspace after base_url= (same connect identity triple as SUMMARY;
// empty string still emits tenant= org= workspace= when unset) so scrapers peer
// ConnectionStatus / SUMMARY base_url, identity and user_agent without inventing
// readiness, result=ok|err after workspace= before exit_code= (derived from failed
// via pullLoopResult; peers ConnectionStatus.Result), and exit_code with the same
// semantics as SUMMARY / process exit (caller passes 1 only when strict && failed;
// otherwise 0). Pure helper (no I/O); version, user_agent, base_url, identity, and
// failed are parameters for testability.
func formatPullLoopResult(exitCode int, failed bool, version, userAgent, baseURL, tenant, org, workspace string) string {
	return fmt.Sprintf("RESULT=done version=%s user_agent=%s base_url=%s tenant=%s org=%s workspace=%s result=%s exit_code=%d", version, userAgent, baseURL, tenant, org, workspace, pullLoopResult(failed), exitCode)
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

// maxWaitReadyMS is the upper clamp for IOMESH_WAIT_READY_MS (2 minutes).
const maxWaitReadyMS = 120000

// defaultWaitIntervalMS is the default WaitReady poll interval when
// IOMESH_WAIT_INTERVAL_MS is empty, invalid, or non-positive.
const defaultWaitIntervalMS = 500

// maxWaitIntervalMS is the upper clamp for IOMESH_WAIT_INTERVAL_MS (1 minute).
const maxWaitIntervalMS = 60000

// parseWaitReadyMS returns the WaitReady budget in milliseconds from an env value.
// Empty, non-positive, or invalid → 0 (skip WaitReady). Values above maxWaitReadyMS
// are clamped. Pure helper (no I/O).
func parseWaitReadyMS(env string) int {
	v := strings.TrimSpace(env)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0
	}
	if n > maxWaitReadyMS {
		return maxWaitReadyMS
	}
	return n
}

// parseWaitIntervalMS returns the WaitReady poll interval in milliseconds from an
// env value. Empty, non-positive, or invalid → 500 (default). Values above
// maxWaitIntervalMS are clamped. Pure helper (no I/O). Only applied when
// IOMESH_WAIT_READY_MS > 0.
func parseWaitIntervalMS(env string) int {
	v := strings.TrimSpace(env)
	if v == "" {
		return defaultWaitIntervalMS
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return defaultWaitIntervalMS
	}
	if n > maxWaitIntervalMS {
		return maxWaitIntervalMS
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
