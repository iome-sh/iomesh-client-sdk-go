package iomeshclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
)

func TestConnectFromEnv_RequiresURL(t *testing.T) {
	_, err := iomeshclient.ConnectFromEnv(map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "IOMESH_URL required") {
		t.Fatalf("empty map err=%v", err)
	}
	_, err = iomeshclient.ConnectFromEnv(map[string]string{"IOMESH_URL": "   "})
	if err == nil || !strings.Contains(err.Error(), "IOMESH_URL required") {
		t.Fatalf("blank URL err=%v", err)
	}
}

func TestConnectFromEnv_EmptyMapIgnoresProcessEnv(t *testing.T) {
	t.Setenv("IOMESH_URL", "http://127.0.0.1:8422")
	_, err := iomeshclient.ConnectFromEnv(map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "IOMESH_URL required") {
		t.Fatalf("non-nil empty map must not read process env, err=%v", err)
	}
}

func TestConnectFromEnv_InvalidTimeout(t *testing.T) {
	_, err := iomeshclient.ConnectFromEnv(map[string]string{
		"IOMESH_URL":     "http://127.0.0.1:8422",
		"IOMESH_TIMEOUT": "not-a-float",
	})
	if err == nil || !strings.Contains(err.Error(), "IOMESH_TIMEOUT invalid") {
		t.Fatalf("err=%v", err)
	}
}

func TestConnectFromEnv_NilUsesProcessEnv(t *testing.T) {
	var gotOrg string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOrg = r.Header.Get("X-IOMesh-Org")
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("IOMESH_URL", srv.URL+"/")
	t.Setenv("IOMESH_ORG", "from-proc")
	nc, err := iomeshclient.ConnectFromEnv(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := nc.Pub(context.Background(), "events.demo", []byte("x"), nil); err != nil {
		t.Fatal(err)
	}
	if gotOrg != "from-proc" {
		t.Fatalf("X-IOMesh-Org=%q", gotOrg)
	}
}

func TestConnectFromEnv_SetsOrgHeaderAndDecodesCatalog(t *testing.T) {
	var mu sync.Mutex
	var gotOrg, gotTenant, gotWS, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotOrg = r.Header.Get("X-IOMesh-Org")
		gotTenant = r.Header.Get("X-IOMesh-Tenant")
		gotWS = r.Header.Get("X-IOMesh-Workspace")
		gotAuth = r.Header.Get("Authorization")
		mu.Unlock()
		if r.Method != http.MethodGet || r.URL.Path != "/v1/streams" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"name":     "EVENTS",
				"org_id":   "acme-org",
				"subjects": []string{"dept.events.>"},
				"messages": 1,
			},
			{
				"name":     "GITHUB_EVENTS",
				"subjects": []string{"github.>"},
				"messages": 2,
			},
		})
	}))
	t.Cleanup(srv.Close)

	nc, err := iomeshclient.ConnectFromEnv(map[string]string{
		"IOMESH_URL":          srv.URL,
		"IOMESH_TENANT":       "dept.engineering",
		"IOMESH_ORG":          "acme-org",
		"IOMESH_WORKSPACE":    "ws_default",
		"IOMESH_BEARER_TOKEN": "secret-token",
		"IOMESH_TIMEOUT":      "12.5",
	})
	if err != nil {
		t.Fatal(err)
	}

	streams, err := nc.ListStreams(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if gotOrg != "acme-org" {
		t.Fatalf("X-IOMesh-Org=%q", gotOrg)
	}
	if gotTenant != "dept.engineering" {
		t.Fatalf("X-IOMesh-Tenant=%q", gotTenant)
	}
	if gotWS != "ws_default" {
		t.Fatalf("X-IOMesh-Workspace=%q", gotWS)
	}
	if gotAuth != "Bearer secret-token" {
		t.Fatalf("Authorization=%q", gotAuth)
	}
	if len(streams) != 2 {
		t.Fatalf("streams=%+v", streams)
	}
	if streams[0].Name != "EVENTS" || streams[0].OrgID != "acme-org" {
		t.Fatalf("owned=%+v", streams[0])
	}
	if streams[1].Name != "GITHUB_EVENTS" || streams[1].OrgID != "" {
		t.Fatalf("shared persist=%+v", streams[1])
	}
}

func TestConnectFromEnv_TokenAliasAndBearerWins(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	nc, err := iomeshclient.ConnectFromEnv(map[string]string{
		"IOMESH_URL":   srv.URL,
		"IOMESH_TOKEN": "legacy-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := nc.Pub(context.Background(), "events.demo", []byte("x"), nil); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer legacy-token" {
		t.Fatalf("legacy Authorization=%q", gotAuth)
	}

	nc2, err := iomeshclient.ConnectFromEnv(map[string]string{
		"IOMESH_URL":          srv.URL,
		"IOMESH_TOKEN":        "legacy-token",
		"IOMESH_BEARER_TOKEN": "primary",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := nc2.Pub(context.Background(), "events.demo", []byte("x"), nil); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer primary" {
		t.Fatalf("primary Authorization=%q", gotAuth)
	}
}

func TestConnectFromEnv_TimeoutHonored(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	nc, err := iomeshclient.ConnectFromEnv(map[string]string{
		"IOMESH_URL":     srv.URL,
		"IOMESH_TIMEOUT": "0.05",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = nc.Pub(context.Background(), "events.demo", []byte("x"), nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
