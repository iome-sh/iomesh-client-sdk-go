package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestListStreamMessages_GitHubReplaySmoke(t *testing.T) {
	t.Parallel()
	payload := base64.StdEncoding.EncodeToString([]byte(`{"ref":"refs/heads/main"}`))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/streams/OPERATIONAL_EVENTS/messages" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]any{
				{
					"stream":    "OPERATIONAL_EVENTS",
					"seq":       4,
					"subject":   "dept.engineering.events.github",
					"partition": 0,
					"payload":   payload,
					"headers":   map[string]string{},
					"timestamp": time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC),
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL}, iomeshclient.WithTenant("dept.engineering"))
	if err != nil {
		t.Fatal(err)
	}
	msgs, err := nc.ListStreamMessages(context.Background(), "OPERATIONAL_EVENTS", iomeshclient.ListStreamMessagesOptions{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	out := formatGitHubReplay("OPERATIONAL_EVENTS", msgs)
	if !strings.Contains(out, "count=1") || !strings.Contains(out, "seq=4") {
		t.Fatalf("httptest replay:\n%s", out)
	}
	if strings.Contains(out, "org-health API") || strings.Contains(out, "heart-rate API") {
		t.Fatalf("must not invent org-health:\n%s", out)
	}
}
