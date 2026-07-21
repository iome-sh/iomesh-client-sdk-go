package main

import (
	"strings"
	"testing"
)

func TestEnvStrict(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		env  string
		want bool
	}{
		{name: "exact 1", env: "1", want: true},
		{name: "empty", env: "", want: false},
		{name: "zero", env: "0", want: false},
		{name: "true string", env: "true", want: false},
		{name: "yes", env: "yes", want: false},
		{name: "whitespace 1", env: " 1 ", want: false},
		{name: "trailing newline", env: "1\n", want: false},
		{name: "TRUE", env: "TRUE", want: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := envStrict(tc.env)
			if got != tc.want {
				t.Fatalf("envStrict(%q) = %v, want %v", tc.env, got, tc.want)
			}
		})
	}
}

func TestEnvWaitRequireHealth(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		env  string
		want bool
	}{
		{name: "exact 1", env: "1", want: true},
		{name: "empty", env: "", want: false},
		{name: "zero", env: "0", want: false},
		{name: "true string", env: "true", want: false},
		{name: "yes", env: "yes", want: false},
		{name: "whitespace 1", env: " 1 ", want: true},
		{name: "trailing newline", env: "1\n", want: true},
		{name: "TRUE", env: "TRUE", want: false},
		{name: "leading tab", env: "\t1", want: true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := envWaitRequireHealth(tc.env)
			if got != tc.want {
				t.Fatalf("envWaitRequireHealth(%q) = %v, want %v", tc.env, got, tc.want)
			}
		})
	}
}

func TestStatusResultFailed(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		result string
		want   bool
	}{
		{name: "err is failed", result: "err", want: true},
		{name: "ok is not failed", result: "ok", want: false},
		{name: "empty is not failed", result: "", want: false},
		{name: "unknown is not failed", result: "unknown", want: false},
		{name: "ERR uppercase is not failed", result: "ERR", want: false},
		{name: "whitespace err is not failed", result: " err ", want: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := statusResultFailed(tc.result)
			if got != tc.want {
				t.Fatalf("statusResultFailed(%q) = %v, want %v", tc.result, got, tc.want)
			}
		})
	}
}

func TestWantPublishEach(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		env  string
		want bool
	}{
		{name: "exact 1", env: "1", want: true},
		{name: "empty", env: "", want: false},
		{name: "zero", env: "0", want: false},
		{name: "true string", env: "true", want: false},
		{name: "yes", env: "yes", want: false},
		{name: "whitespace 1", env: " 1 ", want: false},
		{name: "uppercase", env: "1\n", want: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := wantPublishEach(tc.env)
			if got != tc.want {
				t.Fatalf("wantPublishEach(%q) = %v, want %v", tc.env, got, tc.want)
			}
		})
	}
}

func TestFormatPullLoopSummary(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name              string
		cyclesCompleted   int
		fetchTotal        int
		durationMS        int
		waitReadyMS       int
		waitIntervalMS    int
		waitRequireHealth bool
		waitReadyAttempts int
		failed            bool
		strict            bool
		version           string
		userAgent         string
		baseURL           string
		tenant            string
		org               string
		workspace         string
		stream            string
		consumer          string
		batch             int
		maxWaitMS         int
		loops             int
		want              string
	}{
		{
			name:            "zeroes wait off",
			cyclesCompleted: 0,
			fetchTotal:      0,
			durationMS:      0,
			// wait off: knobs are 0/false even if interval/require/attempts would otherwise be set
			waitReadyMS:       0,
			waitIntervalMS:    500,
			waitRequireHealth: true,
			waitReadyAttempts: 7,
			failed:            false,
			strict:            false,
			version:           "0.51.0",
			want:              "SUMMARY version=0.51.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=0 fetch_total=0 duration_ms=0 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false wait_ready_attempts=0 failed=false strict=false result=ok exit_code=0",
		},
		{
			name:              "typical success wait off",
			cyclesCompleted:   3,
			fetchTotal:        12,
			durationMS:        4500,
			waitReadyMS:       0,
			waitIntervalMS:    500,
			waitRequireHealth: false,
			waitReadyAttempts: 0,
			failed:            false,
			strict:            false,
			version:           "0.51.0",
			want:              "SUMMARY version=0.51.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=3 fetch_total=12 duration_ms=4500 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false wait_ready_attempts=0 failed=false strict=false result=ok exit_code=0",
		},
		{
			name:              "single cycle empty fetch wait off",
			cyclesCompleted:   1,
			fetchTotal:        0,
			durationMS:        2001,
			waitReadyMS:       0,
			waitIntervalMS:    0,
			waitRequireHealth: false,
			waitReadyAttempts: 0,
			failed:            false,
			strict:            false,
			version:           "0.51.0",
			want:              "SUMMARY version=0.51.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=1 fetch_total=0 duration_ms=2001 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false wait_ready_attempts=0 failed=false strict=false result=ok exit_code=0",
		},
		{
			name:              "negative duration clamps to zero",
			cyclesCompleted:   1,
			fetchTotal:        5,
			durationMS:        -10,
			waitReadyMS:       0,
			waitIntervalMS:    250,
			waitRequireHealth: false,
			waitReadyAttempts: 0,
			failed:            false,
			strict:            false,
			version:           "0.51.0",
			want:              "SUMMARY version=0.51.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=1 fetch_total=5 duration_ms=0 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false wait_ready_attempts=0 failed=false strict=false result=ok exit_code=0",
		},
		{
			name:              "wait on default interval require false attempts 1",
			cyclesCompleted:   2,
			fetchTotal:        8,
			durationMS:        1200,
			waitReadyMS:       5000,
			waitIntervalMS:    500,
			waitRequireHealth: false,
			waitReadyAttempts: 1,
			failed:            false,
			strict:            false,
			version:           "0.51.0",
			want:              "SUMMARY version=0.51.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=2 fetch_total=8 duration_ms=1200 wait_ready_ms=5000 wait_interval_ms=500 wait_require_health=false wait_ready_attempts=1 failed=false strict=false result=ok exit_code=0",
		},
		{
			name:              "wait on custom interval require true attempts N",
			cyclesCompleted:   1,
			fetchTotal:        3,
			durationMS:        900,
			waitReadyMS:       3000,
			waitIntervalMS:    250,
			waitRequireHealth: true,
			waitReadyAttempts: 4,
			failed:            false,
			strict:            false,
			version:           "0.51.0",
			want:              "SUMMARY version=0.51.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=1 fetch_total=3 duration_ms=900 wait_ready_ms=3000 wait_interval_ms=250 wait_require_health=true wait_ready_attempts=4 failed=false strict=false result=ok exit_code=0",
		},
		{
			name:              "wait on interval 1 require health attempts 2",
			cyclesCompleted:   0,
			fetchTotal:        0,
			durationMS:        50,
			waitReadyMS:       100,
			waitIntervalMS:    1,
			waitRequireHealth: true,
			waitReadyAttempts: 2,
			failed:            false,
			strict:            false,
			version:           "0.51.0",
			want:              "SUMMARY version=0.51.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=0 fetch_total=0 duration_ms=50 wait_ready_ms=100 wait_interval_ms=1 wait_require_health=true wait_ready_attempts=2 failed=false strict=false result=ok exit_code=0",
		},
		{
			name:              "negative wait ready treated as off zeros attempts",
			cyclesCompleted:   1,
			fetchTotal:        1,
			durationMS:        10,
			waitReadyMS:       -1,
			waitIntervalMS:    500,
			waitRequireHealth: true,
			waitReadyAttempts: 9,
			failed:            false,
			strict:            false,
			version:           "0.51.0",
			want:              "SUMMARY version=0.51.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=1 fetch_total=1 duration_ms=10 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false wait_ready_attempts=0 failed=false strict=false result=ok exit_code=0",
		},
		{
			name:              "failed true strict false exit 0",
			cyclesCompleted:   0,
			fetchTotal:        0,
			durationMS:        100,
			waitReadyMS:       0,
			waitIntervalMS:    500,
			waitRequireHealth: false,
			waitReadyAttempts: 0,
			failed:            true,
			strict:            false,
			version:           "0.51.0",
			want:              "SUMMARY version=0.51.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=0 fetch_total=0 duration_ms=100 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false wait_ready_attempts=0 failed=true strict=false result=err exit_code=0",
		},
		{
			name:              "failed true wait on strict false exit 0",
			cyclesCompleted:   1,
			fetchTotal:        2,
			durationMS:        800,
			waitReadyMS:       2000,
			waitIntervalMS:    250,
			waitRequireHealth: true,
			waitReadyAttempts: 5,
			failed:            true,
			strict:            false,
			version:           "0.51.0",
			want:              "SUMMARY version=0.51.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=1 fetch_total=2 duration_ms=800 wait_ready_ms=2000 wait_interval_ms=250 wait_require_health=true wait_ready_attempts=5 failed=true strict=false result=err exit_code=0",
		},
		{
			name:              "strict true failed false exit 0",
			cyclesCompleted:   2,
			fetchTotal:        4,
			durationMS:        600,
			waitReadyMS:       0,
			waitIntervalMS:    0,
			waitRequireHealth: false,
			waitReadyAttempts: 0,
			failed:            false,
			strict:            true,
			version:           "0.51.0",
			want:              "SUMMARY version=0.51.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=2 fetch_total=4 duration_ms=600 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false wait_ready_attempts=0 failed=false strict=true result=ok exit_code=0",
		},
		{
			name:              "strict true failed true exit 1",
			cyclesCompleted:   1,
			fetchTotal:        0,
			durationMS:        1500,
			waitReadyMS:       5000,
			waitIntervalMS:    500,
			waitRequireHealth: true,
			waitReadyAttempts: 3,
			failed:            true,
			strict:            true,
			version:           "0.51.0",
			want:              "SUMMARY version=0.51.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=1 fetch_total=0 duration_ms=1500 wait_ready_ms=5000 wait_interval_ms=500 wait_require_health=true wait_ready_attempts=3 failed=true strict=true result=err exit_code=1",
		},
		{
			name:              "custom version leading field",
			cyclesCompleted:   1,
			fetchTotal:        0,
			durationMS:        100,
			waitReadyMS:       0,
			waitIntervalMS:    0,
			waitRequireHealth: false,
			waitReadyAttempts: 0,
			failed:            false,
			strict:            false,
			version:           "9.9.9",
			want:              "SUMMARY version=9.9.9 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=1 fetch_total=0 duration_ms=100 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false wait_ready_attempts=0 failed=false strict=false result=ok exit_code=0",
		},
		{
			name:              "empty version still emits version=",
			cyclesCompleted:   0,
			fetchTotal:        0,
			durationMS:        0,
			waitReadyMS:       0,
			waitIntervalMS:    0,
			waitRequireHealth: false,
			waitReadyAttempts: 0,
			failed:            false,
			strict:            false,
			version:           "",
			want:              "SUMMARY version= user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=0 fetch_total=0 duration_ms=0 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false wait_ready_attempts=0 failed=false strict=false result=ok exit_code=0",
		},

		{
			name:              "identity populated after version",
			cyclesCompleted:   1,
			fetchTotal:        0,
			durationMS:        100,
			waitReadyMS:       0,
			waitIntervalMS:    0,
			waitRequireHealth: false,
			waitReadyAttempts: 0,
			failed:            false,
			strict:            false,
			version:           "0.54.0",
			tenant:            "dept.research",
			org:               "org_a",
			workspace:         "ws_1",
			want:              "SUMMARY version=0.54.0 user_agent= base_url= tenant=dept.research org=org_a workspace=ws_1 stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=1 fetch_total=0 duration_ms=100 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false wait_ready_attempts=0 failed=false strict=false result=ok exit_code=0",
		},
		{
			name:              "empty identity still emits tenant= org= workspace=",
			cyclesCompleted:   0,
			fetchTotal:        0,
			durationMS:        0,
			waitReadyMS:       0,
			waitIntervalMS:    0,
			waitRequireHealth: false,
			waitReadyAttempts: 0,
			failed:            false,
			strict:            false,
			version:           "0.54.0",
			tenant:            "",
			org:               "",
			workspace:         "",
			want:              "SUMMARY version=0.54.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=0 fetch_total=0 duration_ms=0 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false wait_ready_attempts=0 failed=false strict=false result=ok exit_code=0",
		},
		{
			name:              "user_agent populated after version before tenant",
			cyclesCompleted:   1,
			fetchTotal:        0,
			durationMS:        100,
			waitReadyMS:       0,
			waitIntervalMS:    0,
			waitRequireHealth: false,
			waitReadyAttempts: 0,
			failed:            false,
			strict:            false,
			version:           "0.57.0",
			userAgent:         "iomesh-client-sdk-go/0.57.0",
			tenant:            "dept.research",
			org:               "org_a",
			workspace:         "ws_1",
			want:              "SUMMARY version=0.57.0 user_agent=iomesh-client-sdk-go/0.57.0 base_url= tenant=dept.research org=org_a workspace=ws_1 stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=1 fetch_total=0 duration_ms=100 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false wait_ready_attempts=0 failed=false strict=false result=ok exit_code=0",
		},
		{
			name:              "empty user_agent still emits user_agent=",
			cyclesCompleted:   0,
			fetchTotal:        0,
			durationMS:        0,
			waitReadyMS:       0,
			waitIntervalMS:    0,
			waitRequireHealth: false,
			waitReadyAttempts: 0,
			failed:            false,
			strict:            false,
			version:           "0.57.0",
			userAgent:         "",
			tenant:            "",
			org:               "",
			workspace:         "",
			want:              "SUMMARY version=0.57.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=0 fetch_total=0 duration_ms=0 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false wait_ready_attempts=0 failed=false strict=false result=ok exit_code=0",
		},
		{
			name:              "base_url populated after user_agent before tenant",
			cyclesCompleted:   1,
			fetchTotal:        0,
			durationMS:        100,
			waitReadyMS:       0,
			waitIntervalMS:    0,
			waitRequireHealth: false,
			waitReadyAttempts: 0,
			failed:            false,
			strict:            false,
			version:           "0.58.0",
			userAgent:         "iomesh-client-sdk-go/0.58.0",
			baseURL:           "http://127.0.0.1:8422",
			tenant:            "dept.research",
			org:               "org_a",
			workspace:         "ws_1",
			want:              "SUMMARY version=0.58.0 user_agent=iomesh-client-sdk-go/0.58.0 base_url=http://127.0.0.1:8422 tenant=dept.research org=org_a workspace=ws_1 stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=1 fetch_total=0 duration_ms=100 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false wait_ready_attempts=0 failed=false strict=false result=ok exit_code=0",
		},
		{
			name:              "empty base_url still emits base_url=",
			cyclesCompleted:   0,
			fetchTotal:        0,
			durationMS:        0,
			waitReadyMS:       0,
			waitIntervalMS:    0,
			waitRequireHealth: false,
			waitReadyAttempts: 0,
			failed:            false,
			strict:            false,
			version:           "0.58.0",
			userAgent:         "",
			baseURL:           "",
			tenant:            "",
			org:               "",
			workspace:         "",
			want:              "SUMMARY version=0.58.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=0 fetch_total=0 duration_ms=0 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false wait_ready_attempts=0 failed=false strict=false result=ok exit_code=0",
		},
		{
			name:              "stream consumer populated after workspace before cycles",
			cyclesCompleted:   1,
			fetchTotal:        0,
			durationMS:        100,
			waitReadyMS:       0,
			waitIntervalMS:    0,
			waitRequireHealth: false,
			waitReadyAttempts: 0,
			failed:            false,
			strict:            false,
			version:           "0.60.0",
			userAgent:         "iomesh-client-sdk-go/0.60.0",
			baseURL:           "http://127.0.0.1:8422",
			tenant:            "dept.research",
			org:               "org_a",
			workspace:         "ws_1",
			stream:            "EVENTS",
			consumer:          "sdk-pull-loop",
			want:              "SUMMARY version=0.60.0 user_agent=iomesh-client-sdk-go/0.60.0 base_url=http://127.0.0.1:8422 tenant=dept.research org=org_a workspace=ws_1 stream=EVENTS consumer=sdk-pull-loop batch=5 max_wait_ms=2000 loops=1 cycles_completed=1 fetch_total=0 duration_ms=100 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false wait_ready_attempts=0 failed=false strict=false result=ok exit_code=0",
		},
		{
			name:              "empty stream consumer still emits stream= consumer=",
			cyclesCompleted:   0,
			fetchTotal:        0,
			durationMS:        0,
			waitReadyMS:       0,
			waitIntervalMS:    0,
			waitRequireHealth: false,
			waitReadyAttempts: 0,
			failed:            false,
			strict:            false,
			version:           "0.60.0",
			userAgent:         "",
			baseURL:           "",
			tenant:            "",
			org:               "",
			workspace:         "",
			stream:            "",
			consumer:          "",
			want:              "SUMMARY version=0.60.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=0 fetch_total=0 duration_ms=0 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false wait_ready_attempts=0 failed=false strict=false result=ok exit_code=0",
		},
		{
			name:              "non-default batch max_wait_ms loops after consumer",
			cyclesCompleted:   3,
			fetchTotal:        12,
			durationMS:        4500,
			waitReadyMS:       0,
			waitIntervalMS:    0,
			waitRequireHealth: false,
			waitReadyAttempts: 0,
			failed:            false,
			strict:            false,
			version:           "0.61.0",
			userAgent:         "iomesh-client-sdk-go/0.61.0",
			baseURL:           "http://127.0.0.1:8422",
			tenant:            "dept.research",
			org:               "org_a",
			workspace:         "ws_1",
			stream:            "EVENTS",
			consumer:          "sdk-pull-loop",
			batch:             10,
			maxWaitMS:         500,
			loops:             3,
			want:              "SUMMARY version=0.61.0 user_agent=iomesh-client-sdk-go/0.61.0 base_url=http://127.0.0.1:8422 tenant=dept.research org=org_a workspace=ws_1 stream=EVENTS consumer=sdk-pull-loop batch=10 max_wait_ms=500 loops=3 cycles_completed=3 fetch_total=12 duration_ms=4500 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false wait_ready_attempts=0 failed=false strict=false result=ok exit_code=0",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Prefer table fields; when all three are zero (legacy cases), use defaults 5/2000/1.
			batch, maxWaitMS, loops := tc.batch, tc.maxWaitMS, tc.loops
			if tc.batch == 0 && tc.maxWaitMS == 0 && tc.loops == 0 {
				batch, maxWaitMS, loops = 5, 2000, 1
			}
			got := formatPullLoopSummary(tc.cyclesCompleted, tc.fetchTotal, tc.durationMS, tc.waitReadyMS, tc.waitIntervalMS, tc.waitRequireHealth, tc.waitReadyAttempts, tc.failed, tc.strict, tc.version, tc.userAgent, tc.baseURL, tc.tenant, tc.org, tc.workspace, tc.stream, tc.consumer, batch, maxWaitMS, loops)
			if got != tc.want {
				t.Fatalf("formatPullLoopSummary(...) = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatPullLoopSummaryAlwaysEmitsIdentity(t *testing.T) {
	t.Parallel()
	// Empty identity still emits keys (honest empty strings).
	got := formatPullLoopSummary(0, 0, 0, 0, 0, false, 0, false, false, "0.54.0", "", "", "", "", "", "", "", 5, 2000, 1)
	for _, key := range []string{"tenant=", "org=", "workspace="} {
		if !strings.Contains(got, key) {
			t.Fatalf("missing %s in %q", key, got)
		}
	}
	if !strings.Contains(got, "version=0.54.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=") {
		t.Fatalf("identity order want after version/user_agent/base_url: %q", got)
	}
	// Populated identity passes through; does not invent readiness / exit success.
	got2 := formatPullLoopSummary(1, 0, 10, 0, 0, false, 0, true, true, "0.54.0", "iomesh-client-sdk-go/0.54.0", "", "dept.x", "org_a", "ws_y", "", "", 5, 2000, 1)
	if !strings.Contains(got2, "tenant=dept.x") || !strings.Contains(got2, "org=org_a") || !strings.Contains(got2, "workspace=ws_y") {
		t.Fatalf("populated identity missing: %q", got2)
	}
	if !strings.Contains(got2, "failed=true") || !strings.Contains(got2, "result=err") || !strings.Contains(got2, "exit_code=1") {
		t.Fatalf("identity must not invent success: %q", got2)
	}
}

func TestFormatPullLoopSummaryAlwaysEmitsUserAgent(t *testing.T) {
	t.Parallel()
	// Empty user_agent still emits key (honest empty string).
	got := formatPullLoopSummary(0, 0, 0, 0, 0, false, 0, false, false, "0.57.0", "", "", "", "", "", "", "", 5, 2000, 1)
	if !strings.Contains(got, "user_agent=") {
		t.Fatalf("missing user_agent= in %q", got)
	}
	if !strings.Contains(got, "version=0.57.0 user_agent= base_url= tenant=") {
		t.Fatalf("user_agent order want after version before base_url: %q", got)
	}
	// Package-default UA passes through; does not invent readiness / exit success.
	ua := "iomesh-client-sdk-go/0.57.0"
	got2 := formatPullLoopSummary(1, 0, 10, 0, 0, false, 0, true, true, "0.57.0", ua, "", "dept.x", "org_a", "ws_y", "", "", 5, 2000, 1)
	if !strings.Contains(got2, "user_agent="+ua) {
		t.Fatalf("populated user_agent missing: %q", got2)
	}
	if !strings.Contains(got2, "version=0.57.0 user_agent="+ua+" base_url= tenant=dept.x") {
		t.Fatalf("user_agent order want after version before base_url/tenant: %q", got2)
	}
	if !strings.Contains(got2, "failed=true") || !strings.Contains(got2, "result=err") || !strings.Contains(got2, "exit_code=1") {
		t.Fatalf("user_agent must not invent success: %q", got2)
	}
}

func TestFormatPullLoopSummaryAlwaysEmitsBaseURL(t *testing.T) {
	t.Parallel()
	// Empty base_url still emits key (honest empty string).
	got := formatPullLoopSummary(0, 0, 0, 0, 0, false, 0, false, false, "0.58.0", "", "", "", "", "", "", "", 5, 2000, 1)
	if !strings.Contains(got, "base_url=") {
		t.Fatalf("missing base_url= in %q", got)
	}
	if !strings.Contains(got, "version=0.58.0 user_agent= base_url= tenant=") {
		t.Fatalf("base_url order want after user_agent before tenant: %q", got)
	}
	// Connect mesh URL passes through; does not invent readiness / exit success.
	ua := "iomesh-client-sdk-go/0.58.0"
	base := "http://127.0.0.1:8422"
	got2 := formatPullLoopSummary(1, 0, 10, 0, 0, false, 0, true, true, "0.58.0", ua, base, "dept.x", "org_a", "ws_y", "", "", 5, 2000, 1)
	if !strings.Contains(got2, "base_url="+base) {
		t.Fatalf("populated base_url missing: %q", got2)
	}
	if !strings.Contains(got2, "version=0.58.0 user_agent="+ua+" base_url="+base+" tenant=dept.x") {
		t.Fatalf("base_url order want after user_agent before tenant: %q", got2)
	}
	if !strings.Contains(got2, "failed=true") || !strings.Contains(got2, "result=err") || !strings.Contains(got2, "exit_code=1") {
		t.Fatalf("base_url must not invent success: %q", got2)
	}
}

func TestFormatPullLoopSummaryAlwaysEmitsStreamConsumer(t *testing.T) {
	t.Parallel()
	// Empty stream/consumer still emit keys (honest empty strings).
	got := formatPullLoopSummary(0, 0, 0, 0, 0, false, 0, false, false, "0.60.0", "", "", "", "", "", "", "", 5, 2000, 1)
	for _, key := range []string{"stream=", "consumer="} {
		if !strings.Contains(got, key) {
			t.Fatalf("missing %s in %q", key, got)
		}
	}
	if !strings.Contains(got, "workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 cycles_completed=") {
		t.Fatalf("stream/consumer order want after workspace before cycles_completed: %q", got)
	}
	// Connect config passes through; does not invent readiness / exit success.
	ua := "iomesh-client-sdk-go/0.60.0"
	base := "http://127.0.0.1:8422"
	got2 := formatPullLoopSummary(1, 0, 10, 0, 0, false, 0, true, true, "0.60.0", ua, base, "dept.x", "org_a", "ws_y", "EVENTS", "sdk-pull-loop", 5, 2000, 1)
	if !strings.Contains(got2, "stream=EVENTS") || !strings.Contains(got2, "consumer=sdk-pull-loop") {
		t.Fatalf("populated stream/consumer missing: %q", got2)
	}
	if !strings.Contains(got2, "workspace=ws_y stream=EVENTS consumer=sdk-pull-loop batch=5 max_wait_ms=2000 loops=1 cycles_completed=") {
		t.Fatalf("stream/consumer order want after workspace before cycles_completed: %q", got2)
	}
	if !strings.Contains(got2, "failed=true") || !strings.Contains(got2, "result=err") || !strings.Contains(got2, "exit_code=1") {
		t.Fatalf("stream/consumer must not invent success: %q", got2)
	}
}

func TestFormatPullLoopResult(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		exitCode  int
		failed    bool
		version   string
		userAgent string
		baseURL   string
		tenant    string
		org       string
		workspace string
		stream    string
		consumer  string
		batch     int
		maxWaitMS int
		loops     int
		want      string
	}{
		{name: "exit 0 success empty identity", exitCode: 0, failed: false, version: "0.52.0", want: "RESULT=done version=0.52.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=ok exit_code=0"},
		{name: "exit 1 strict failed empty identity", exitCode: 1, failed: true, version: "0.52.0", want: "RESULT=done version=0.52.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=err exit_code=1"},
		// same matrix as SUMMARY: scrapers pass the computed code; helper formats only
		{name: "non-strict failed still 0 result err", exitCode: 0, failed: true, version: "0.52.0", want: "RESULT=done version=0.52.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=err exit_code=0"},
		{name: "strict ok still 0 result ok", exitCode: 0, failed: false, version: "0.52.0", want: "RESULT=done version=0.52.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=ok exit_code=0"},
		{name: "custom version", exitCode: 0, failed: false, version: "9.9.9", want: "RESULT=done version=9.9.9 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=ok exit_code=0"},
		{name: "empty version still emits version=", exitCode: 0, failed: false, version: "", want: "RESULT=done version= user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=ok exit_code=0"},
		{name: "empty version with exit 1", exitCode: 1, failed: true, version: "", want: "RESULT=done version= user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=err exit_code=1"},
		{
			name: "identity populated after version", exitCode: 0, failed: false, version: "0.56.0",
			tenant: "dept.research", org: "org_a", workspace: "ws_1",
			want: "RESULT=done version=0.56.0 user_agent= base_url= tenant=dept.research org=org_a workspace=ws_1 stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=ok exit_code=0",
		},
		{
			name: "empty identity still emits tenant= org= workspace=", exitCode: 0, failed: false, version: "0.56.0",
			tenant: "", org: "", workspace: "",
			want: "RESULT=done version=0.56.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=ok exit_code=0",
		},
		{
			name: "populated identity with exit 1 does not invent success", exitCode: 1, failed: true, version: "0.56.0",
			tenant: "dept.x", org: "org_a", workspace: "ws_y",
			want: "RESULT=done version=0.56.0 user_agent= base_url= tenant=dept.x org=org_a workspace=ws_y stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=err exit_code=1",
		},
		{
			name: "user_agent populated after version before tenant", exitCode: 0, failed: false, version: "0.57.0",
			userAgent: "iomesh-client-sdk-go/0.57.0",
			tenant:    "dept.research", org: "org_a", workspace: "ws_1",
			want: "RESULT=done version=0.57.0 user_agent=iomesh-client-sdk-go/0.57.0 base_url= tenant=dept.research org=org_a workspace=ws_1 stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=ok exit_code=0",
		},
		{
			name: "empty user_agent still emits user_agent=", exitCode: 0, failed: false, version: "0.57.0",
			userAgent: "", tenant: "", org: "", workspace: "",
			want: "RESULT=done version=0.57.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=ok exit_code=0",
		},
		{
			name: "populated user_agent with exit 1 does not invent success", exitCode: 1, failed: true, version: "0.57.0",
			userAgent: "iomesh-client-sdk-go/0.57.0",
			tenant:    "dept.x", org: "org_a", workspace: "ws_y",
			want: "RESULT=done version=0.57.0 user_agent=iomesh-client-sdk-go/0.57.0 base_url= tenant=dept.x org=org_a workspace=ws_y stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=err exit_code=1",
		},
		{
			name: "base_url populated after user_agent before tenant", exitCode: 0, failed: false, version: "0.58.0",
			userAgent: "iomesh-client-sdk-go/0.58.0",
			baseURL:   "http://127.0.0.1:8422",
			tenant:    "dept.research", org: "org_a", workspace: "ws_1",
			want: "RESULT=done version=0.58.0 user_agent=iomesh-client-sdk-go/0.58.0 base_url=http://127.0.0.1:8422 tenant=dept.research org=org_a workspace=ws_1 stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=ok exit_code=0",
		},
		{
			name: "empty base_url still emits base_url=", exitCode: 0, failed: false, version: "0.58.0",
			userAgent: "", baseURL: "", tenant: "", org: "", workspace: "",
			want: "RESULT=done version=0.58.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=ok exit_code=0",
		},
		{
			name: "populated base_url with exit 1 does not invent success", exitCode: 1, failed: true, version: "0.58.0",
			userAgent: "iomesh-client-sdk-go/0.58.0",
			baseURL:   "http://127.0.0.1:8422",
			tenant:    "dept.x", org: "org_a", workspace: "ws_y",
			want: "RESULT=done version=0.58.0 user_agent=iomesh-client-sdk-go/0.58.0 base_url=http://127.0.0.1:8422 tenant=dept.x org=org_a workspace=ws_y stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=err exit_code=1",
		},
		{
			name: "stream consumer populated after workspace before result", exitCode: 0, failed: false, version: "0.60.0",
			userAgent: "iomesh-client-sdk-go/0.60.0",
			baseURL:   "http://127.0.0.1:8422",
			tenant:    "dept.research", org: "org_a", workspace: "ws_1",
			stream: "EVENTS", consumer: "sdk-pull-loop",
			want: "RESULT=done version=0.60.0 user_agent=iomesh-client-sdk-go/0.60.0 base_url=http://127.0.0.1:8422 tenant=dept.research org=org_a workspace=ws_1 stream=EVENTS consumer=sdk-pull-loop batch=5 max_wait_ms=2000 loops=1 result=ok exit_code=0",
		},
		{
			name: "empty stream consumer still emits stream= consumer=", exitCode: 0, failed: false, version: "0.60.0",
			userAgent: "", baseURL: "", tenant: "", org: "", workspace: "", stream: "", consumer: "",
			want: "RESULT=done version=0.60.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=ok exit_code=0",
		},
		{
			name: "populated stream consumer with exit 1 does not invent success", exitCode: 1, failed: true, version: "0.60.0",
			userAgent: "iomesh-client-sdk-go/0.60.0",
			baseURL:   "http://127.0.0.1:8422",
			tenant:    "dept.x", org: "org_a", workspace: "ws_y",
			stream: "ORDERS", consumer: "pull-worker",
			want: "RESULT=done version=0.60.0 user_agent=iomesh-client-sdk-go/0.60.0 base_url=http://127.0.0.1:8422 tenant=dept.x org=org_a workspace=ws_y stream=ORDERS consumer=pull-worker batch=5 max_wait_ms=2000 loops=1 result=err exit_code=1",
		},
		{
			name: "non-default batch max_wait_ms loops after consumer", exitCode: 0, failed: false, version: "0.61.0",
			userAgent: "iomesh-client-sdk-go/0.61.0",
			baseURL:   "http://127.0.0.1:8422",
			tenant:    "dept.research", org: "org_a", workspace: "ws_1",
			stream: "EVENTS", consumer: "sdk-pull-loop",
			batch: 10, maxWaitMS: 500, loops: 3,
			want: "RESULT=done version=0.61.0 user_agent=iomesh-client-sdk-go/0.61.0 base_url=http://127.0.0.1:8422 tenant=dept.research org=org_a workspace=ws_1 stream=EVENTS consumer=sdk-pull-loop batch=10 max_wait_ms=500 loops=3 result=ok exit_code=0",
		},
		{
			name: "non-default knobs with exit 1 does not invent success", exitCode: 1, failed: true, version: "0.61.0",
			userAgent: "iomesh-client-sdk-go/0.61.0",
			baseURL:   "http://127.0.0.1:8422",
			tenant:    "dept.x", org: "org_a", workspace: "ws_y",
			stream: "ORDERS", consumer: "pull-worker",
			batch: 10, maxWaitMS: 500, loops: 3,
			want: "RESULT=done version=0.61.0 user_agent=iomesh-client-sdk-go/0.61.0 base_url=http://127.0.0.1:8422 tenant=dept.x org=org_a workspace=ws_y stream=ORDERS consumer=pull-worker batch=10 max_wait_ms=500 loops=3 result=err exit_code=1",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Prefer table fields; when all three are zero (legacy cases), use defaults 5/2000/1.
			batch, maxWaitMS, loops := tc.batch, tc.maxWaitMS, tc.loops
			if tc.batch == 0 && tc.maxWaitMS == 0 && tc.loops == 0 {
				batch, maxWaitMS, loops = 5, 2000, 1
			}
			got := formatPullLoopResult(tc.exitCode, tc.failed, tc.version, tc.userAgent, tc.baseURL, tc.tenant, tc.org, tc.workspace, tc.stream, tc.consumer, batch, maxWaitMS, loops)
			if got != tc.want {
				t.Fatalf("formatPullLoopResult(...) = %q, want %q", got, tc.want)
			}
			if !strings.Contains(got, "version=") {
				t.Fatalf("formatPullLoopResult(...) = %q, want always contains version=", got)
			}
			for _, key := range []string{"user_agent=", "base_url=", "tenant=", "org=", "workspace=", "stream=", "consumer=", "batch=", "max_wait_ms=", "loops=", "result=", "exit_code="} {
				if !strings.Contains(got, key) {
					t.Fatalf("formatPullLoopResult(...) = %q, want always contains %s", got, key)
				}
			}
		})
	}
}

func TestFormatPullLoopResultAlwaysEmitsIdentity(t *testing.T) {
	t.Parallel()
	// Empty identity still emits keys (honest empty strings).
	got := formatPullLoopResult(0, false, "0.56.0", "", "", "", "", "", "", "", 5, 2000, 1)
	for _, key := range []string{"tenant=", "org=", "workspace="} {
		if !strings.Contains(got, key) {
			t.Fatalf("missing %s in %q", key, got)
		}
	}
	if !strings.Contains(got, "version=0.56.0 user_agent= base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=ok exit_code=") {
		t.Fatalf("identity order want after version/user_agent/base_url: %q", got)
	}
	// Populated identity passes through; does not invent readiness / exit success.
	got2 := formatPullLoopResult(1, true, "0.56.0", "iomesh-client-sdk-go/0.56.0", "", "dept.x", "org_a", "ws_y", "", "", 5, 2000, 1)
	if !strings.Contains(got2, "tenant=dept.x") || !strings.Contains(got2, "org=org_a") || !strings.Contains(got2, "workspace=ws_y") {
		t.Fatalf("populated identity missing: %q", got2)
	}
	if !strings.Contains(got2, "result=err") || !strings.Contains(got2, "exit_code=1") {
		t.Fatalf("identity must not invent success: %q", got2)
	}
}

func TestFormatPullLoopResultAlwaysEmitsUserAgent(t *testing.T) {
	t.Parallel()
	// Empty user_agent still emits key (honest empty string).
	got := formatPullLoopResult(0, false, "0.57.0", "", "", "", "", "", "", "", 5, 2000, 1)
	if !strings.Contains(got, "user_agent=") {
		t.Fatalf("missing user_agent= in %q", got)
	}
	if !strings.Contains(got, "version=0.57.0 user_agent= base_url= tenant=") {
		t.Fatalf("user_agent order want after version before base_url: %q", got)
	}
	// Package-default UA passes through; does not invent readiness / exit success.
	ua := "iomesh-client-sdk-go/0.57.0"
	got2 := formatPullLoopResult(1, true, "0.57.0", ua, "", "dept.x", "org_a", "ws_y", "", "", 5, 2000, 1)
	if !strings.Contains(got2, "user_agent="+ua) {
		t.Fatalf("populated user_agent missing: %q", got2)
	}
	if !strings.Contains(got2, "version=0.57.0 user_agent="+ua+" base_url= tenant=dept.x") {
		t.Fatalf("user_agent order want after version before base_url/tenant: %q", got2)
	}
	if !strings.Contains(got2, "result=err") || !strings.Contains(got2, "exit_code=1") {
		t.Fatalf("user_agent must not invent success: %q", got2)
	}
}

func TestFormatPullLoopResultAlwaysEmitsBaseURL(t *testing.T) {
	t.Parallel()
	// Empty base_url still emits key (honest empty string).
	got := formatPullLoopResult(0, false, "0.58.0", "", "", "", "", "", "", "", 5, 2000, 1)
	if !strings.Contains(got, "base_url=") {
		t.Fatalf("missing base_url= in %q", got)
	}
	if !strings.Contains(got, "version=0.58.0 user_agent= base_url= tenant=") {
		t.Fatalf("base_url order want after user_agent before tenant: %q", got)
	}
	// Connect mesh URL passes through; does not invent readiness / exit success.
	ua := "iomesh-client-sdk-go/0.58.0"
	base := "http://127.0.0.1:8422"
	got2 := formatPullLoopResult(1, true, "0.58.0", ua, base, "dept.x", "org_a", "ws_y", "", "", 5, 2000, 1)
	if !strings.Contains(got2, "base_url="+base) {
		t.Fatalf("populated base_url missing: %q", got2)
	}
	if !strings.Contains(got2, "version=0.58.0 user_agent="+ua+" base_url="+base+" tenant=dept.x") {
		t.Fatalf("base_url order want after user_agent before tenant: %q", got2)
	}
	if !strings.Contains(got2, "result=err") || !strings.Contains(got2, "exit_code=1") {
		t.Fatalf("base_url must not invent success: %q", got2)
	}
}

func TestFormatPullLoopResultAlwaysEmitsStreamConsumer(t *testing.T) {
	t.Parallel()
	// Empty stream/consumer still emit keys (honest empty strings).
	got := formatPullLoopResult(0, false, "0.60.0", "", "", "", "", "", "", "", 5, 2000, 1)
	for _, key := range []string{"stream=", "consumer="} {
		if !strings.Contains(got, key) {
			t.Fatalf("missing %s in %q", key, got)
		}
	}
	if !strings.Contains(got, "workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=ok exit_code=") {
		t.Fatalf("stream/consumer order want after workspace before result: %q", got)
	}
	// Connect config passes through; does not invent readiness / exit success.
	ua := "iomesh-client-sdk-go/0.60.0"
	base := "http://127.0.0.1:8422"
	got2 := formatPullLoopResult(1, true, "0.60.0", ua, base, "dept.x", "org_a", "ws_y", "EVENTS", "sdk-pull-loop", 5, 2000, 1)
	if !strings.Contains(got2, "stream=EVENTS") || !strings.Contains(got2, "consumer=sdk-pull-loop") {
		t.Fatalf("populated stream/consumer missing: %q", got2)
	}
	if !strings.Contains(got2, "workspace=ws_y stream=EVENTS consumer=sdk-pull-loop batch=5 max_wait_ms=2000 loops=1 result=err exit_code=1") {
		t.Fatalf("stream/consumer order want after workspace before result: %q", got2)
	}
	if !strings.Contains(got2, "result=err") || !strings.Contains(got2, "exit_code=1") {
		t.Fatalf("stream/consumer must not invent success: %q", got2)
	}
}

func TestFormatPullLoopSummaryAlwaysEmitsBatchMaxWaitLoops(t *testing.T) {
	t.Parallel()
	// Zero knobs still emit keys (honest zeros; clamp is at parse time, not formatter).
	got := formatPullLoopSummary(0, 0, 0, 0, 0, false, 0, false, false, "0.61.0", "", "", "", "", "", "", "", 0, 0, 0)
	for _, key := range []string{"batch=", "max_wait_ms=", "loops="} {
		if !strings.Contains(got, key) {
			t.Fatalf("missing %s in %q", key, got)
		}
	}
	if !strings.Contains(got, "consumer= batch=0 max_wait_ms=0 loops=0 cycles_completed=") {
		t.Fatalf("batch/max_wait_ms/loops order want after consumer before cycles_completed: %q", got)
	}
	// Non-default knobs pass through; does not invent readiness / exit success.
	ua := "iomesh-client-sdk-go/0.61.0"
	base := "http://127.0.0.1:8422"
	got2 := formatPullLoopSummary(1, 0, 10, 0, 0, false, 0, true, true, "0.61.0", ua, base, "dept.x", "org_a", "ws_y", "EVENTS", "sdk-pull-loop", 10, 500, 3)
	if !strings.Contains(got2, "batch=10") || !strings.Contains(got2, "max_wait_ms=500") || !strings.Contains(got2, "loops=3") {
		t.Fatalf("populated batch/max_wait_ms/loops missing: %q", got2)
	}
	if !strings.Contains(got2, "consumer=sdk-pull-loop batch=10 max_wait_ms=500 loops=3 cycles_completed=") {
		t.Fatalf("batch/max_wait_ms/loops order want after consumer before cycles_completed: %q", got2)
	}
	if !strings.Contains(got2, "failed=true") || !strings.Contains(got2, "result=err") || !strings.Contains(got2, "exit_code=1") {
		t.Fatalf("batch/max_wait_ms/loops must not invent success: %q", got2)
	}
}

func TestFormatPullLoopResultAlwaysEmitsBatchMaxWaitLoops(t *testing.T) {
	t.Parallel()
	// Zero knobs still emit keys (honest zeros; clamp is at parse time, not formatter).
	got := formatPullLoopResult(0, false, "0.61.0", "", "", "", "", "", "", "", 0, 0, 0)
	for _, key := range []string{"batch=", "max_wait_ms=", "loops="} {
		if !strings.Contains(got, key) {
			t.Fatalf("missing %s in %q", key, got)
		}
	}
	if !strings.Contains(got, "consumer= batch=0 max_wait_ms=0 loops=0 result=ok exit_code=") {
		t.Fatalf("batch/max_wait_ms/loops order want after consumer before result: %q", got)
	}
	// Non-default knobs pass through; does not invent readiness / exit success.
	ua := "iomesh-client-sdk-go/0.61.0"
	base := "http://127.0.0.1:8422"
	got2 := formatPullLoopResult(1, true, "0.61.0", ua, base, "dept.x", "org_a", "ws_y", "EVENTS", "sdk-pull-loop", 10, 500, 3)
	if !strings.Contains(got2, "batch=10") || !strings.Contains(got2, "max_wait_ms=500") || !strings.Contains(got2, "loops=3") {
		t.Fatalf("populated batch/max_wait_ms/loops missing: %q", got2)
	}
	if !strings.Contains(got2, "consumer=sdk-pull-loop batch=10 max_wait_ms=500 loops=3 result=err exit_code=1") {
		t.Fatalf("batch/max_wait_ms/loops order want after consumer before result: %q", got2)
	}
	if !strings.Contains(got2, "result=err") || !strings.Contains(got2, "exit_code=1") {
		t.Fatalf("batch/max_wait_ms/loops must not invent success: %q", got2)
	}
}

// TestFormatPullLoopResultExitCodeMatrix covers the strict×failed → exit_code
// matrix used by printPullLoopDone (same rule as SUMMARY / process exit).
// version, user_agent, base_url, identity, and result are always emitted alongside exit_code.
func TestFormatPullLoopResultExitCodeMatrix(t *testing.T) {
	t.Parallel()
	const ver = "0.52.0"
	const ua = "iomesh-client-sdk-go/0.52.0"
	cases := []struct {
		name   string
		failed bool
		strict bool
		want   string
	}{
		{name: "ok non-strict", failed: false, strict: false, want: "RESULT=done version=0.52.0 user_agent=iomesh-client-sdk-go/0.52.0 base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=ok exit_code=0"},
		{name: "failed non-strict", failed: true, strict: false, want: "RESULT=done version=0.52.0 user_agent=iomesh-client-sdk-go/0.52.0 base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=err exit_code=0"},
		{name: "ok strict", failed: false, strict: true, want: "RESULT=done version=0.52.0 user_agent=iomesh-client-sdk-go/0.52.0 base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=ok exit_code=0"},
		{name: "failed strict", failed: true, strict: true, want: "RESULT=done version=0.52.0 user_agent=iomesh-client-sdk-go/0.52.0 base_url= tenant= org= workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=err exit_code=1"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			exitCode := 0
			if tc.strict && tc.failed {
				exitCode = 1
			}
			got := formatPullLoopResult(exitCode, tc.failed, ver, ua, "", "", "", "", "", "", 5, 2000, 1)
			if got != tc.want {
				t.Fatalf("formatPullLoopResult(strict=%v failed=%v) = %q, want %q", tc.strict, tc.failed, got, tc.want)
			}
			if !strings.Contains(got, "version=") {
				t.Fatalf("formatPullLoopResult(strict=%v failed=%v) = %q, want always contains version=", tc.strict, tc.failed, got)
			}
			for _, key := range []string{"user_agent=", "base_url=", "tenant=", "org=", "workspace=", "stream=", "consumer=", "batch=", "max_wait_ms=", "loops=", "result=", "exit_code="} {
				if !strings.Contains(got, key) {
					t.Fatalf("formatPullLoopResult(strict=%v failed=%v) = %q, want always contains %s", tc.strict, tc.failed, got, key)
				}
			}
		})
	}
}

// TestPullLoopResult maps failed → result=ok|err (peers ConnectionStatus.Result).
func TestPullLoopResult(t *testing.T) {
	t.Parallel()
	if got := pullLoopResult(false); got != "ok" {
		t.Fatalf("pullLoopResult(false) = %q, want ok", got)
	}
	if got := pullLoopResult(true); got != "err" {
		t.Fatalf("pullLoopResult(true) = %q, want err", got)
	}
}

// TestFormatPullLoopSummaryAlwaysEmitsResult covers result=ok|err from failed on SUMMARY.
func TestFormatPullLoopSummaryAlwaysEmitsResult(t *testing.T) {
	t.Parallel()
	ok := formatPullLoopSummary(0, 0, 0, 0, 0, false, 0, false, false, "0.59.0", "", "", "", "", "", "", "", 5, 2000, 1)
	if !strings.Contains(ok, "failed=false strict=false result=ok exit_code=0") {
		t.Fatalf("want result=ok after strict before exit_code: %q", ok)
	}
	errLine := formatPullLoopSummary(1, 0, 10, 0, 0, false, 0, true, false, "0.59.0", "", "", "", "", "", "", "", 5, 2000, 1)
	if !strings.Contains(errLine, "failed=true strict=false result=err exit_code=0") {
		t.Fatalf("want result=err when failed (non-strict exit 0): %q", errLine)
	}
	strictErr := formatPullLoopSummary(1, 0, 10, 0, 0, false, 0, true, true, "0.59.0", "", "", "", "", "", "", "", 5, 2000, 1)
	if !strings.Contains(strictErr, "failed=true strict=true result=err exit_code=1") {
		t.Fatalf("want result=err when failed strict: %q", strictErr)
	}
}

// TestFormatPullLoopResultAlwaysEmitsResult covers result=ok|err from failed on RESULT.
func TestFormatPullLoopResultAlwaysEmitsResult(t *testing.T) {
	t.Parallel()
	ok := formatPullLoopResult(0, false, "0.59.0", "", "", "", "", "", "", "", 5, 2000, 1)
	if !strings.Contains(ok, "workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=ok exit_code=0") {
		t.Fatalf("want result=ok before exit_code: %q", ok)
	}
	// Non-strict failed: exit_code=0 but result=err (honest hard-fail flag).
	errLine := formatPullLoopResult(0, true, "0.59.0", "", "", "", "", "", "", "", 5, 2000, 1)
	if !strings.Contains(errLine, "workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=err exit_code=0") {
		t.Fatalf("want result=err when failed (non-strict exit 0): %q", errLine)
	}
	strictErr := formatPullLoopResult(1, true, "0.59.0", "", "", "", "", "", "", "", 5, 2000, 1)
	if !strings.Contains(strictErr, "workspace= stream= consumer= batch=5 max_wait_ms=2000 loops=1 result=err exit_code=1") {
		t.Fatalf("want result=err when failed strict: %q", strictErr)
	}
}

func TestParseLoops(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		env  string
		def  int
		want int
	}{
		{name: "empty uses default 1", env: "", def: 1, want: 1},
		{name: "whitespace uses default", env: "  ", def: 1, want: 1},
		{name: "invalid uses default", env: "abc", def: 1, want: 1},
		{name: "explicit 1", env: "1", def: 1, want: 1},
		{name: "explicit 3", env: "3", def: 1, want: 3},
		{name: "zero clamps to 1", env: "0", def: 1, want: 1},
		{name: "negative clamps to 1", env: "-5", def: 1, want: 1},
		{name: "above max clamps to 100", env: "101", def: 1, want: 100},
		{name: "max 100", env: "100", def: 1, want: 100},
		{name: "default below 1 clamps", env: "", def: 0, want: 1},
		{name: "default above 100 clamps", env: "", def: 200, want: 100},
		{name: "trim digits", env: "  7  ", def: 1, want: 7},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseLoops(tc.env, tc.def)
			if got != tc.want {
				t.Fatalf("parseLoops(%q, %d) = %d, want %d", tc.env, tc.def, got, tc.want)
			}
		})
	}
}

func TestParseWaitReadyMS(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		env  string
		want int
	}{
		{name: "empty is zero", env: "", want: 0},
		{name: "whitespace is zero", env: "  ", want: 0},
		{name: "invalid is zero", env: "abc", want: 0},
		{name: "zero is zero", env: "0", want: 0},
		{name: "negative is zero", env: "-1", want: 0},
		{name: "explicit 5000", env: "5000", want: 5000},
		{name: "explicit 1", env: "1", want: 1},
		{name: "max 120000", env: "120000", want: 120000},
		{name: "above max clamps", env: "120001", want: 120000},
		{name: "large clamps", env: "999999999", want: 120000},
		{name: "trim digits", env: "  7500  ", want: 7500},
		{name: "float invalid", env: "1.5", want: 0},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseWaitReadyMS(tc.env)
			if got != tc.want {
				t.Fatalf("parseWaitReadyMS(%q) = %d, want %d", tc.env, got, tc.want)
			}
		})
	}
}

func TestParseWaitIntervalMS(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		env  string
		want int
	}{
		{name: "empty defaults 500", env: "", want: 500},
		{name: "whitespace defaults 500", env: "  ", want: 500},
		{name: "invalid defaults 500", env: "abc", want: 500},
		{name: "zero defaults 500", env: "0", want: 500},
		{name: "negative defaults 500", env: "-1", want: 500},
		{name: "explicit 250", env: "250", want: 250},
		{name: "explicit 1", env: "1", want: 1},
		{name: "explicit 500", env: "500", want: 500},
		{name: "max 60000", env: "60000", want: 60000},
		{name: "above max clamps", env: "60001", want: 60000},
		{name: "huge clamps", env: "999999999", want: 60000},
		{name: "trim digits", env: "  250  ", want: 250},
		{name: "float invalid defaults 500", env: "1.5", want: 500},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseWaitIntervalMS(tc.env)
			if got != tc.want {
				t.Fatalf("parseWaitIntervalMS(%q) = %d, want %d", tc.env, got, tc.want)
			}
		})
	}
}

func TestResolveConsumerFilter(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		subjectEnv   string
		ensureStream bool
		want         string
	}{
		{
			name:       "explicit subject wins",
			subjectEnv: "tenant.events",
			want:       "tenant.events",
		},
		{
			name:         "explicit subject wins even when ensure on",
			subjectEnv:   "tenant.events",
			ensureStream: true,
			want:         "tenant.events",
		},
		{
			name:         "ensure defaults to stream.>",
			ensureStream: true,
			want:         "stream.>",
		},
		{
			name: "empty without ensure",
			want: "",
		},
		{
			name:         "trim subject",
			subjectEnv:   "  orders.>  ",
			ensureStream: true,
			want:         "orders.>",
		},
		{
			name:         "whitespace-only subject falls through to ensure default",
			subjectEnv:   "   ",
			ensureStream: true,
			want:         "stream.>",
		},
		{
			name:       "whitespace-only subject without ensure is empty",
			subjectEnv: "   ",
			want:       "",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := resolveConsumerFilter(tc.subjectEnv, tc.ensureStream)
			if got != tc.want {
				t.Fatalf("resolveConsumerFilter(%q, %v) = %q, want %q", tc.subjectEnv, tc.ensureStream, got, tc.want)
			}
		})
	}
}

func TestResolvePublishSubject(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		pubSubject   string
		filter       string
		tenant       string
		stream       string
		ensureStream bool
		want         string
	}{
		{
			name:       "explicit pub subject wins",
			pubSubject: "custom.subject",
			filter:     "filter.x",
			tenant:     "demo.tenant",
			stream:     "EVENTS",
			want:       "custom.subject",
		},
		{
			name:         "explicit pub wins over ensure default",
			pubSubject:   "stream.custom",
			ensureStream: true,
			tenant:       "demo.tenant",
			stream:       "EVENTS",
			want:         "stream.custom",
		},
		{
			name:         "filter wins even when ensure on and not under stream.",
			filter:       "tenant.events",
			tenant:       "demo.tenant",
			stream:       "EVENTS",
			ensureStream: true,
			want:         "tenant.events",
		},
		{
			name:         "ensure defaults under stream.>",
			tenant:       "demo.tenant",
			stream:       "EVENTS",
			ensureStream: true,
			want:         "stream.sdk-pull-loop",
		},
		{
			name:   "tenant default without ensure",
			tenant: "demo.tenant",
			stream: "EVENTS",
			want:   "demo.tenant.sdk-pull-loop",
		},
		{
			name:   "stream demo when no tenant",
			stream: "EVENTS",
			want:   "EVENTS.demo",
		},
		{
			name:         "ensure ignores tenant for default",
			tenant:       "other.tenant",
			stream:       "OTHER",
			ensureStream: true,
			want:         "stream.sdk-pull-loop",
		},
		{
			name:       "trim pub subject",
			pubSubject: "  stream.x  ",
			want:       "stream.x",
		},
		{
			name:       "empty pub falls through to filter",
			pubSubject: "   ",
			filter:     "a.b",
			want:       "a.b",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := resolvePublishSubject(tc.pubSubject, tc.filter, tc.tenant, tc.stream, tc.ensureStream)
			if got != tc.want {
				t.Fatalf("resolvePublishSubject(...) = %q, want %q", got, tc.want)
			}
		})
	}
}
