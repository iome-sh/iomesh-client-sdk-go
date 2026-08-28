package main

import (
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
)

func TestFormatGitHubReplay_EmptyIsHonest(t *testing.T) {
	t.Parallel()
	out := formatGitHubReplay("OPERATIONAL_EVENTS", nil)
	for _, n := range []string{
		"count=0",
		"no signed GitHub event",
		"not org-health",
		"not heart-rate",
		"Slack/PagerDuty not pulses here",
		"chat is not the record",
		"catalog ≠ Connected",
	} {
		if !strings.Contains(out, n) {
			t.Fatalf("missing %q\n%s", n, out)
		}
	}
	for _, bad := range []string{"org-health API", "heart-rate API", "PagerDuty pulse", "Connected workspace"} {
		if strings.Contains(out, bad) {
			t.Fatalf("must not invent %q\n%s", bad, out)
		}
	}
}

func TestFormatGitHubReplay_OneGitHubEvent(t *testing.T) {
	t.Parallel()
	out := formatGitHubReplay("OPERATIONAL_EVENTS", []iomeshclient.StreamMessage{
		{Seq: 4, Subject: "dept.engineering.events.github", Payload: []byte(`{"ref":"refs/heads/main"}`)},
	})
	if !strings.Contains(out, "count=1") || !strings.Contains(out, "seq=4") {
		t.Fatalf("want one github row:\n%s", out)
	}
	if !strings.Contains(out, "dept.engineering.events.github") {
		t.Fatalf("want github subject:\n%s", out)
	}
	if strings.Contains(out, "no signed GitHub event") {
		t.Fatalf("populated replay must not print empty honesty:\n%s", out)
	}
}
