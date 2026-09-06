package iomeshclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
)

func TestRequireOrg_FailsClosedOnFetchAndCatalogWithoutOrg(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("must not call broker when org is required and empty")
	}))
	t.Cleanup(srv.Close)

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL}, iomeshclient.WithRequireOrg())
	if err != nil {
		t.Fatal(err)
	}
	if !nc.RequireOrg() {
		t.Fatal("RequireOrg() = false, want true")
	}

	_, err = nc.ConsumerFetch(context.Background(), "EVENTS", "worker", 1)
	if err == nil || !strings.Contains(err.Error(), "X-IOMesh-Org required") {
		t.Fatalf("ConsumerFetch err=%v", err)
	}
	_, err = nc.ListStreams(context.Background())
	if err == nil || !strings.Contains(err.Error(), "X-IOMesh-Org required") {
		t.Fatalf("ListStreams err=%v", err)
	}
	err = nc.ConsumerAck(context.Background(), "EVENTS", "worker", 1)
	if err == nil || !strings.Contains(err.Error(), "X-IOMesh-Org required") {
		t.Fatalf("ConsumerAck err=%v", err)
	}
	_, err = nc.ListStreamMessages(context.Background(), "OPERATIONAL_EVENTS", iomeshclient.ListStreamMessagesOptions{Limit: 1})
	if err == nil || !strings.Contains(err.Error(), "X-IOMesh-Org required") {
		t.Fatalf("ListStreamMessages err=%v", err)
	}
}

func TestRequireOrg_FetchSendsHeaderWhenSet(t *testing.T) {
	var gotOrg string
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		gotOrg = r.Header.Get("X-IOMesh-Org")
		_ = json.NewEncoder(w).Encode(map[string]any{"messages": []any{}})
	}))
	t.Cleanup(srv.Close)

	nc, err := iomeshclient.Connect(
		iomeshclient.Options{URL: srv.URL},
		iomeshclient.WithOrg("org_a"),
		iomeshclient.WithRequireOrg(),
	)
	if err != nil {
		t.Fatal(err)
	}
	msgs, err := nc.ConsumerFetch(context.Background(), "EVENTS", "worker", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 0 {
		t.Fatalf("msgs=%d", len(msgs))
	}
	if hits != 1 {
		t.Fatalf("hits=%d", hits)
	}
	if gotOrg != "org_a" {
		t.Fatalf("X-IOMesh-Org=%q", gotOrg)
	}
}

func TestRequireOrg_OffOmitsHeaderAndStillFetches(t *testing.T) {
	var gotOrg string
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		gotOrg = r.Header.Get("X-IOMesh-Org")
		_ = json.NewEncoder(w).Encode(map[string]any{"messages": []any{}})
	}))
	t.Cleanup(srv.Close)

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if nc.RequireOrg() {
		t.Fatal("RequireOrg() default must be false")
	}
	if _, err := nc.ConsumerFetch(context.Background(), "EVENTS", "worker", 1); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("hits=%d want 1 (fail-open still reaches broker)", hits)
	}
	if gotOrg != "" {
		t.Fatalf("X-IOMesh-Org=%q, want omitted", gotOrg)
	}
}

func TestConnectFromEnv_RequireOrgFlag(t *testing.T) {
	nc, err := iomeshclient.ConnectFromEnv(map[string]string{
		"IOMESH_URL":         "http://127.0.0.1:9",
		"IOMESH_REQUIRE_ORG": "1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !nc.RequireOrg() {
		t.Fatal("IOMESH_REQUIRE_ORG=1 must set RequireOrg")
	}

	ncTrue, err := iomeshclient.ConnectFromEnv(map[string]string{
		"IOMESH_URL":         "http://127.0.0.1:9",
		"IOMESH_REQUIRE_ORG": "TRUE",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ncTrue.RequireOrg() {
		t.Fatal("IOMESH_REQUIRE_ORG=TRUE must set RequireOrg")
	}

	ncOff, err := iomeshclient.ConnectFromEnv(map[string]string{
		"IOMESH_URL": "http://127.0.0.1:9",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ncOff.RequireOrg() {
		t.Fatal("default ConnectFromEnv must leave RequireOrg off")
	}
}

func TestRequireOrg_DoesNotGateHealth(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Path != "/health" && r.URL.Path != "/v1/health" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL}, iomeshclient.WithRequireOrg())
	if err != nil {
		t.Fatal(err)
	}
	if err := nc.Health(context.Background()); err != nil {
		t.Fatalf("Health must still run when RequireOrg is on: %v", err)
	}
	if hits == 0 {
		t.Fatal("Health must reach the broker")
	}
}
