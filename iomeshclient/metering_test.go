package iomeshclient_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
)

func TestEmitLLMCall_PublishWireAndHeaders(t *testing.T) {
	var mu sync.Mutex
	var gotPath, gotSubject, gotOrg, gotWS string
	var decoded map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/streams/dept/publish", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		gotPath = r.URL.Path
		gotOrg = r.Header.Get("X-IOMesh-Org")
		gotWS = r.Header.Get("X-IOMesh-Workspace")
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotSubject, _ = body["subject"].(string)
		if s, ok := body["payload"].(string); ok {
			raw, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				t.Errorf("payload b64: %v", err)
			} else {
				_ = json.Unmarshal(raw, &decoded)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"stream": "dept", "seq": 9, "subject": gotSubject,
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	nc, err := iomeshclient.Connect(
		iomeshclient.Options{URL: ts.URL},
		iomeshclient.WithTenant("dept.research"),
		iomeshclient.WithOrg("org_a"),
		iomeshclient.WithWorkspace("ws_1"),
	)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}

	ack, err := nc.EmitLLMCall(context.Background(), iomeshclient.LLMCallEvent{
		Tenant:       "dept.research",
		SessionID:    "sess-1",
		Model:        "deepseek-v4-flash",
		ModelID:      "deepseek-v4-flash",
		DurationMS:   12,
		Attempts:     1,
		EstUSD:       0.001,
		PromptTokens: 5,
		TotalTokens:  10,
	})
	if err != nil {
		t.Fatalf("EmitLLMCall: %v", err)
	}
	if ack == nil || ack.Seq != 9 {
		t.Fatalf("ack=%+v", ack)
	}

	mu.Lock()
	defer mu.Unlock()
	if gotPath != "/v1/streams/dept/publish" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotSubject != "dept.agent.llm_call" {
		t.Fatalf("subject=%q", gotSubject)
	}
	if gotOrg != "org_a" || gotWS != "ws_1" {
		t.Fatalf("headers org=%q ws=%q", gotOrg, gotWS)
	}
	if decoded["type"] != "dept.agent.llm_call" {
		t.Fatalf("type=%v", decoded["type"])
	}
	if decoded["session_id"] != "sess-1" {
		t.Fatalf("session_id=%v", decoded["session_id"])
	}
	payload, _ := decoded["payload"].(map[string]any)
	if payload["model"] != "deepseek-v4-flash" {
		t.Fatalf("payload=%v", payload)
	}
	if payload["org"] != "org_a" || payload["workspace"] != "ws_1" {
		t.Fatalf("payload multi-tenant=%v", payload)
	}
	tokens, _ := payload["tokens"].(map[string]any)
	if tokens["total"] != float64(10) {
		t.Fatalf("tokens=%v", tokens)
	}
}

func TestEmitDeptEvent_RequiresType(t *testing.T) {
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: "http://127.0.0.1:9"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = nc.EmitDeptEvent(context.Background(), iomeshclient.DeptEvent{})
	if err == nil || !strings.Contains(err.Error(), "type required") {
		t.Fatalf("err=%v", err)
	}
}
