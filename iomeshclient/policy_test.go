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

func TestEvaluatePolicy_ModeOff(t *testing.T) {
	// No server needed — mode off short-circuits.
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: "http://127.0.0.1:1"}, iomeshclient.WithTenant("t"))
	if err != nil {
		t.Fatal(err)
	}
	dec := nc.EvaluatePolicy(context.Background(), iomeshclient.PolicyInput{
		Tool: "run_shell",
		Mode: iomeshclient.PolicyOff,
	})
	if !dec.Allow || dec.Source != "off" || dec.Mode != iomeshclient.PolicyOff {
		t.Fatalf("%+v", dec)
	}
	if dec.ShouldBlockTool() {
		t.Fatal("off must not block")
	}

	// Empty mode normalizes to off.
	dec2 := nc.EvaluatePolicy(context.Background(), iomeshclient.PolicyInput{Tool: "x"})
	if !dec2.Allow || dec2.Source != "off" {
		t.Fatalf("%+v", dec2)
	}
}

func TestEvaluatePolicy_EnforceDenyMesh(t *testing.T) {
	var gotPath, gotUA string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotUA = r.Header.Get("User-Agent")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"allow":   false,
			"reasons": []string{"rego: shell blocked"},
		})
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL}, iomeshclient.WithTenant("t"))
	if err != nil {
		t.Fatal(err)
	}
	dec := nc.EvaluatePolicy(context.Background(), iomeshclient.PolicyInput{
		Tool: "run_shell",
		Mode: iomeshclient.PolicyEnforce,
	})
	if dec.Allow || !dec.ShouldBlockTool() || dec.Source != "mesh" {
		t.Fatalf("%+v", dec)
	}
	if gotPath != "/v1/policy/evaluate" {
		t.Fatalf("path=%q", gotPath)
	}
	if !strings.Contains(gotUA, "iomesh-client-sdk-go") {
		t.Fatalf("User-Agent=%q", gotUA)
	}
	if !strings.HasPrefix(gotUA, "iomesh-client-sdk-go/") {
		t.Fatalf("default UA prefix missing: %q", gotUA)
	}
	if gotBody["tool"] != "run_shell" {
		t.Fatalf("body tool=%v", gotBody["tool"])
	}
	if gotBody["action"] != "tool.run_shell" {
		t.Fatalf("body action=%v (auto action expected)", gotBody["action"])
	}
	if gotBody["tenant"] != "t" {
		t.Fatalf("body tenant=%v", gotBody["tenant"])
	}
	if gotBody["mode"] != "enforce" {
		t.Fatalf("body mode=%v", gotBody["mode"])
	}
	if !strings.Contains(dec.Summary(), "deny") {
		t.Fatalf("summary=%s", dec.Summary())
	}
}

func TestEvaluatePolicy_AdvisoryDeny(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"allow": false, "reason": "nope"})
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL}, iomeshclient.WithTenant("t"))
	if err != nil {
		t.Fatal(err)
	}
	dec := nc.EvaluatePolicy(context.Background(), iomeshclient.PolicyInput{
		Tool: "write_file",
		Mode: iomeshclient.PolicyAdvisory,
	})
	if dec.Allow {
		t.Fatal("mesh said deny")
	}
	if dec.Source != "mesh" {
		t.Fatalf("source=%s", dec.Source)
	}
	if dec.ShouldBlockTool() {
		t.Fatal("advisory must not ShouldBlockTool")
	}
}

func TestEvaluatePolicy_404Unavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL}, iomeshclient.WithTenant("t"))
	if err != nil {
		t.Fatal(err)
	}
	dec := nc.EvaluatePolicy(context.Background(), iomeshclient.PolicyInput{
		Tool: "x",
		Mode: iomeshclient.PolicyEnforce,
	})
	if !dec.Allow || dec.Source != "unavailable" || dec.ShouldBlockTool() {
		t.Fatalf("%+v", dec)
	}
}

func TestEvaluatePolicy_UnreachableFailOpen(t *testing.T) {
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: "http://127.0.0.1:1"}, iomeshclient.WithTenant("t"))
	if err != nil {
		t.Fatal(err)
	}
	dec := nc.EvaluatePolicy(context.Background(), iomeshclient.PolicyInput{
		Action: "tool.run_shell",
		Tool:   "run_shell",
		Mode:   iomeshclient.PolicyEnforce,
	})
	if !dec.Allow || dec.Source != "fail-open" {
		t.Fatalf("%+v", dec)
	}
	if dec.ShouldBlockTool() {
		t.Fatal("fail-open must not block")
	}
}

func TestEvaluatePolicy_NilClient(t *testing.T) {
	var nc *iomeshclient.Client
	dec := nc.EvaluatePolicy(context.Background(), iomeshclient.PolicyInput{
		Tool: "x",
		Mode: iomeshclient.PolicyEnforce,
	})
	if !dec.Allow || dec.Source != "fail-open" {
		t.Fatalf("%+v", dec)
	}
	if len(dec.Reasons) == 0 || !strings.Contains(dec.Reasons[0], "nil client") {
		t.Fatalf("reasons=%v", dec.Reasons)
	}

	decOff := nc.EvaluatePolicy(context.Background(), iomeshclient.PolicyInput{Mode: iomeshclient.PolicyOff})
	if !decOff.Allow || decOff.Source != "off" {
		t.Fatalf("%+v", decOff)
	}
}
