package main

import "testing"

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
		failed            bool
		want              string
	}{
		{
			name:            "zeroes wait off",
			cyclesCompleted: 0,
			fetchTotal:      0,
			durationMS:      0,
			// wait off: knobs are 0/false even if interval/require would otherwise be set
			waitReadyMS:       0,
			waitIntervalMS:    500,
			waitRequireHealth: true,
			failed:            false,
			want:              "SUMMARY cycles_completed=0 fetch_total=0 duration_ms=0 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false failed=false",
		},
		{
			name:              "typical success wait off",
			cyclesCompleted:   3,
			fetchTotal:        12,
			durationMS:        4500,
			waitReadyMS:       0,
			waitIntervalMS:    500,
			waitRequireHealth: false,
			failed:            false,
			want:              "SUMMARY cycles_completed=3 fetch_total=12 duration_ms=4500 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false failed=false",
		},
		{
			name:              "single cycle empty fetch wait off",
			cyclesCompleted:   1,
			fetchTotal:        0,
			durationMS:        2001,
			waitReadyMS:       0,
			waitIntervalMS:    0,
			waitRequireHealth: false,
			failed:            false,
			want:              "SUMMARY cycles_completed=1 fetch_total=0 duration_ms=2001 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false failed=false",
		},
		{
			name:              "negative duration clamps to zero",
			cyclesCompleted:   1,
			fetchTotal:        5,
			durationMS:        -10,
			waitReadyMS:       0,
			waitIntervalMS:    250,
			waitRequireHealth: false,
			failed:            false,
			want:              "SUMMARY cycles_completed=1 fetch_total=5 duration_ms=0 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false failed=false",
		},
		{
			name:              "wait on default interval require false",
			cyclesCompleted:   2,
			fetchTotal:        8,
			durationMS:        1200,
			waitReadyMS:       5000,
			waitIntervalMS:    500,
			waitRequireHealth: false,
			failed:            false,
			want:              "SUMMARY cycles_completed=2 fetch_total=8 duration_ms=1200 wait_ready_ms=5000 wait_interval_ms=500 wait_require_health=false failed=false",
		},
		{
			name:              "wait on custom interval require true",
			cyclesCompleted:   1,
			fetchTotal:        3,
			durationMS:        900,
			waitReadyMS:       3000,
			waitIntervalMS:    250,
			waitRequireHealth: true,
			failed:            false,
			want:              "SUMMARY cycles_completed=1 fetch_total=3 duration_ms=900 wait_ready_ms=3000 wait_interval_ms=250 wait_require_health=true failed=false",
		},
		{
			name:              "wait on interval 1 require health",
			cyclesCompleted:   0,
			fetchTotal:        0,
			durationMS:        50,
			waitReadyMS:       100,
			waitIntervalMS:    1,
			waitRequireHealth: true,
			failed:            false,
			want:              "SUMMARY cycles_completed=0 fetch_total=0 duration_ms=50 wait_ready_ms=100 wait_interval_ms=1 wait_require_health=true failed=false",
		},
		{
			name:              "negative wait ready treated as off",
			cyclesCompleted:   1,
			fetchTotal:        1,
			durationMS:        10,
			waitReadyMS:       -1,
			waitIntervalMS:    500,
			waitRequireHealth: true,
			failed:            false,
			want:              "SUMMARY cycles_completed=1 fetch_total=1 duration_ms=10 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false failed=false",
		},
		{
			name:              "failed true wait off",
			cyclesCompleted:   0,
			fetchTotal:        0,
			durationMS:        100,
			waitReadyMS:       0,
			waitIntervalMS:    500,
			waitRequireHealth: false,
			failed:            true,
			want:              "SUMMARY cycles_completed=0 fetch_total=0 duration_ms=100 wait_ready_ms=0 wait_interval_ms=0 wait_require_health=false failed=true",
		},
		{
			name:              "failed true wait on",
			cyclesCompleted:   1,
			fetchTotal:        2,
			durationMS:        800,
			waitReadyMS:       2000,
			waitIntervalMS:    250,
			waitRequireHealth: true,
			failed:            true,
			want:              "SUMMARY cycles_completed=1 fetch_total=2 duration_ms=800 wait_ready_ms=2000 wait_interval_ms=250 wait_require_health=true failed=true",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := formatPullLoopSummary(tc.cyclesCompleted, tc.fetchTotal, tc.durationMS, tc.waitReadyMS, tc.waitIntervalMS, tc.waitRequireHealth, tc.failed)
			if got != tc.want {
				t.Fatalf("formatPullLoopSummary(...) = %q, want %q", got, tc.want)
			}
		})
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
