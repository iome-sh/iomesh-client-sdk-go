package main

import "testing"

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
