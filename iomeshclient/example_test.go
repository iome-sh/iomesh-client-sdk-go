package iomeshclient_test

import (
	"fmt"
	"time"

	"github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
)

// ExampleFormatStreams shows compact table output after ListStreams (no network).
// Live usage: streams, err := nc.ListStreams(ctx); fmt.Print(FormatStreams(streams)).
func ExampleFormatStreams() {
	streams := []iomeshclient.StreamInfo{
		{
			Name:       "EVENTS",
			Subjects:   []string{"dept.events.>"},
			Retention:  "limits",
			Partitions: 1,
			Messages:   3,
			FirstSeq:   1,
			LastSeq:    3,
		},
	}
	fmt.Print(iomeshclient.FormatStreams(streams))
	// Output:
	// iomesh streams count=1
	// NAME                         MSGS    FIRST     LAST  PART RETENTION  SUBJECTS
	// EVENTS                          3        1        3     1 limits     dept.events.>
}

// ExampleFormatStreamDetail shows multi-line detail after GetStream (no network).
// Live usage: info, err := nc.GetStream(ctx, "EVENTS"); fmt.Print(FormatStreamDetail(*info)).
func ExampleFormatStreamDetail() {
	max := int64(1000)
	age := int64(3600)
	info := iomeshclient.StreamInfo{
		Name:        "EVENTS",
		Description: "ops",
		Retention:   "limits",
		Partitions:  1,
		MaxMsgs:     &max,
		MaxAgeSec:   &age,
		Messages:    10,
		FirstSeq:    1,
		LastSeq:     10,
		CreatedAt:   time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		Subjects:    []string{"dept.events.>"},
	}
	fmt.Print(iomeshclient.FormatStreamDetail(info))
	// Output:
	// iomesh stream
	// name:        EVENTS
	// description: ops
	// retention:   limits
	// partitions:  1
	// max_msgs:    1000
	// max_age_sec: 3600
	// messages:    10
	// first_seq:   1
	// last_seq:    10
	// created_at:  2026-07-01T12:00:00Z
	// subjects:
	//   - dept.events.>
}

// ExampleFormatConnectionStatus shows operator diagnostics after ConnectionStatus (no network).
// Live usage: st := nc.ConnectionStatus(ctx); fmt.Print(FormatConnectionStatus(st)).
func ExampleFormatConnectionStatus() {
	st := iomeshclient.ConnectionStatus{
		BaseURL:   "http://127.0.0.1:8422",
		UserAgent: "iomesh-client-sdk-go/0.26.0",
		Version:   "0.26.0",
		HealthOK:  true,
		ReadyOK:   true,
	}
	fmt.Print(iomeshclient.FormatConnectionStatus(st))
	// Output:
	// base_url=http://127.0.0.1:8422
	// tenant=
	// org=
	// workspace=
	// user_agent=iomesh-client-sdk-go/0.26.0
	// version=0.26.0
	// health=ok
	// health_err=
	// health_ms=0
	// ready=ok
	// ready_err=
	// ready_ms=0
	// duration_ms=0
	// result=ok
}

// ExampleFormatContextSnippet shows prompt injection formatting after QueryContext (no network).
func ExampleFormatContextSnippet() {
	res := iomeshclient.ContextResult{
		Text: "hello mesh",
		Lineage: []iomeshclient.LineageRef{
			{ID: "dp-1", Subject: "events.>", Source: "catalog"},
		},
	}
	fmt.Print(iomeshclient.FormatContextSnippet(res))
	// Output:
	// hello mesh
	//
	// <iomesh-lineage>
	// - dp-1 · subject=events.> · source=catalog
	// </iomesh-lineage>
}

// ExamplePolicyDecision_Summary shows operator-facing policy summary (no network).
// Live usage: dec := nc.EvaluatePolicy(ctx, PolicyInput{Tool: "run_shell", Mode: PolicyEnforce}).
func ExamplePolicyDecision_Summary() {
	off := iomeshclient.PolicyDecision{Allow: true, Source: "off", Mode: iomeshclient.PolicyOff}
	fmt.Println(off.Summary())

	deny := iomeshclient.PolicyDecision{
		Allow:   false,
		Source:  "mesh",
		Mode:    iomeshclient.PolicyEnforce,
		Reasons: []string{"deny tool"},
	}
	fmt.Println(deny.Summary())
	// Output:
	// allow source=off mode=off
	// deny source=mesh mode=enforce reasons=deny tool
}

// ExamplePolicyDecision_ShouldBlockTool shows enforce-only blocking (no network).
func ExamplePolicyDecision_ShouldBlockTool() {
	// off / fail-open never blocks tools
	off := iomeshclient.PolicyDecision{Allow: true, Source: "off", Mode: iomeshclient.PolicyOff}
	fmt.Println(off.ShouldBlockTool())

	// mesh deny under enforce blocks
	deny := iomeshclient.PolicyDecision{
		Allow:  false,
		Source: "mesh",
		Mode:   iomeshclient.PolicyEnforce,
	}
	fmt.Println(deny.ShouldBlockTool())
	// Output:
	// false
	// true
}

// ExampleFormatKVKeys shows compact key listing after ListKeys (no network).
// Live usage: keys, err := nc.ListKeys(ctx, "agent-state", "worker-"); fmt.Print(FormatKVKeys("agent-state", keys)).
func ExampleFormatKVKeys() {
	keys := []string{"worker-1.checkpoint", "worker-2.checkpoint"}
	fmt.Print(iomeshclient.FormatKVKeys("agent-state", keys))
	// Output:
	// iomesh kv keys bucket=agent-state count=2
	// worker-1.checkpoint
	// worker-2.checkpoint
}

// ExampleFormatBucketInfo shows multi-line bucket detail after CreateBucket / EnsureBucket (no network).
// Live usage: info, err := nc.EnsureBucket(ctx, "agent-state"); fmt.Print(FormatBucketInfo(*info)).
func ExampleFormatBucketInfo() {
	maxBytes := int64(1024)
	ttl := int64(3600)
	info := iomeshclient.BucketInfo{
		Name:       "agent-state",
		History:    5,
		MaxBytes:   &maxBytes,
		TTLSeconds: &ttl,
	}
	fmt.Print(iomeshclient.FormatBucketInfo(info))
	// Output:
	// iomesh kv bucket
	// name:         agent-state
	// history:      5
	// max_bytes:    1024
	// ttl_seconds:  3600
}

// Example_streamLifecycle documents CreateStream / ListStreams / GetStream call shape.
// Network omitted: format helpers only (examples stay deterministic for godoc).
func Example_streamLifecycle() {
	// Create / ensure (returns *StreamInfo on 201 or best-effort GET on 409):
	//   info, err := nc.CreateStream(ctx, iomeshclient.StreamConfig{
	//       Name: "EVENTS", Subjects: []string{"dept.events.>"},
	//   })
	//   info, err = nc.EnsureStream(ctx, cfg) // same semantics
	//
	// List / get:
	//   streams, err := nc.ListStreams(ctx)
	//   info, err := nc.GetStream(ctx, "EVENTS")

	// Format for operators (pure, no I/O):
	streams := []iomeshclient.StreamInfo{
		{Name: "EVENTS", Messages: 0, Subjects: []string{"dept.events.>"}},
	}
	fmt.Print(iomeshclient.FormatStreams(streams))
	// Output:
	// iomesh streams count=1
	// NAME                         MSGS    FIRST     LAST  PART RETENTION  SUBJECTS
	// EVENTS                          0        0        0     0            dept.events.>
}

// ExampleFormatMsgs shows the empty-batch header for operator logs (no network).
// Live usage: batch, err := sub.FetchContext(ctx, 10); fmt.Print(FormatMsgs(batch)).
func ExampleFormatMsgs() {
	fmt.Print(iomeshclient.FormatMsgs(nil))
	// Output:
	// iomesh msgs count=0
}

// Example_pullLoop documents the durable pull fetch → format → ack pattern.
// Network omitted: call shape only (examples stay deterministic for godoc).
func Example_pullLoop() {
	// sub, err := nc.PullSubscribe(ctx, iomeshclient.PullSubscribeConfig{
	//     Stream: "EVENTS", Consumer: "worker-1", Filter: "dept.events.>",
	// })
	// for {
	//     batch, err := sub.FetchContext(ctx, 10, iomeshclient.MaxWait(2*time.Second))
	//     if err != nil { … }
	//     if len(batch) == 0 { continue } // idle long-poll
	//     fmt.Print(iomeshclient.FormatMsgs(batch))
	//     seqs := make([]uint64, len(batch))
	//     for i, m := range batch { seqs[i] = m.Seq() }
	//     if err := sub.AckContext(ctx, seqs...); err != nil { … }
	// }

	// Format empty batch (pure, no I/O):
	fmt.Print(iomeshclient.FormatMsgs(nil))
	// Output:
	// iomesh msgs count=0
}
