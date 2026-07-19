// Command memory-metering-dogfood is a stage smoke for public SDK memory + metering wire.
//
// Env:
//
//	IOMESH_URL            mesh broker base (required)
//	IOMESH_TENANT         tenant (default demo.tenant)
//	IOMESH_ORG            optional X-IOMesh-Org
//	IOMESH_WORKSPACE      optional X-IOMesh-Workspace
//	IOMESH_API_KEY        optional Bearer
//	IOMESH_MEMORY_ENDPOINT optional memory sidecar base for sync retrieve
//	                       (when unset, uses IOMESH_URL — broker-only often 404s retrieve)
//	IOMESH_POLICY_MODE    optional off|advisory|enforce (default off); when advisory/enforce,
//	                       probes EvaluatePolicy for tool.run_shell (warn-only, never exits)
//	IOMESH_WAIT_READY     set to 1 to poll WaitReady before continuing (default: single Ready)
//
// Usage:
//
//	export IOMESH_URL=http://127.0.0.1:8422
//	export IOMESH_MEMORY_ENDPOINT=http://127.0.0.1:8765
//	export IOMESH_POLICY_MODE=advisory   # optional
//	export IOMESH_WAIT_READY=1          # optional
//	go run ./examples/memory-metering-dogfood
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
)

func main() {
	base := env("IOMESH_URL", "")
	if base == "" {
		log.Fatal("IOMESH_URL required")
	}
	tenant := env("IOMESH_TENANT", "demo.tenant")
	memoryBase := env("IOMESH_MEMORY_ENDPOINT", base)

	opts := []iomeshclient.ConnectOpt{
		iomeshclient.WithTenant(tenant),
	}
	if org := os.Getenv("IOMESH_ORG"); org != "" {
		opts = append(opts, iomeshclient.WithOrg(org))
	}
	if ws := os.Getenv("IOMESH_WORKSPACE"); ws != "" {
		opts = append(opts, iomeshclient.WithWorkspace(ws))
	}
	if key := os.Getenv("IOMESH_API_KEY"); key != "" {
		opts = append(opts, iomeshclient.WithBearerToken(key))
	}

	mesh, err := iomeshclient.Connect(iomeshclient.Options{URL: base}, opts...)
	if err != nil {
		log.Fatalf("mesh connect: %v", err)
	}
	memory, err := iomeshclient.Connect(iomeshclient.Options{URL: memoryBase}, opts...)
	if err != nil {
		log.Fatalf("memory connect: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	sessionID := tenant + ".sdk-dogfood"
	now := time.Now().UTC().Format(time.RFC3339)

	fmt.Printf("sdk=%s user-agent=iomesh-client-sdk-go/%s\n", iomeshclient.Version, iomeshclient.Version)

	// 0) Health / ready (mesh broker)
	if err := mesh.Health(ctx); err != nil {
		log.Printf("WARN Health: %v", err)
	} else {
		fmt.Println("PASS Health GET /health")
	}
	if os.Getenv("IOMESH_WAIT_READY") == "1" {
		waitCtx, waitCancel := context.WithTimeout(ctx, 15*time.Second)
		err := mesh.WaitReady(waitCtx, iomeshclient.WaitReadyOptions{Interval: 500 * time.Millisecond})
		waitCancel()
		if err != nil {
			log.Printf("WARN WaitReady: %v", err)
		} else {
			fmt.Println("PASS WaitReady")
		}
	} else if err := mesh.Ready(ctx); err != nil {
		log.Printf("WARN Ready: %v (optional on some brokers)", err)
	} else {
		fmt.Println("PASS Ready GET /ready|/readyz")
	}

	// 0a) Catalog plane (fail-open; warn-only)
	cat := mesh.ListCatalog(ctx, "")
	if cat.Source == "fail-open" || cat.Source == "off" {
		log.Printf("WARN ListCatalog source=%s detail=%s", cat.Source, cat.Detail)
	} else {
		fmt.Printf("PASS ListCatalog source=%s products=%d detail=%s\n", cat.Source, len(cat.Products), cat.Detail)
	}

	// 0b) Optional policy evaluate (fail-open; warn-only)
	policyMode := strings.ToLower(strings.TrimSpace(os.Getenv("IOMESH_POLICY_MODE")))
	if policyMode == "advisory" || policyMode == "enforce" {
		dec := mesh.EvaluatePolicy(ctx, iomeshclient.PolicyInput{
			Tool: "run_shell",
			Mode: iomeshclient.PolicyMode(policyMode),
		})
		log.Printf("policy: %s", dec.Summary())
		fmt.Printf("PASS EvaluatePolicy mode=%s source=%s allow=%v\n", dec.Mode, dec.Source, dec.Allow)
	}

	// 1) Dual-write: async MEMORY_INGEST + optional sync Palace ingest (fail-open)
	env := iomeshclient.MemoryEnvelope{
		Role:       "tool",
		Content:    "iomesh-client-sdk-go memory-metering-dogfood",
		SessionID:  sessionID,
		SessionSeq: 1,
		EventTime:  now,
	}
	dw, err := mesh.DualWriteMemoryTurn(ctx, tenant, env, iomeshclient.DualWriteMemoryOptions{
		Sync:       memoryBase != base,
		SyncClient: memory,
	})
	if err != nil {
		log.Printf("WARN DualWriteMemoryTurn async: %v", err)
	} else {
		fmt.Printf("PASS DualWriteMemoryTurn async_seq=%d", dw.Async.Seq)
		if memoryBase != base {
			if dw.SyncErr != nil {
				fmt.Printf(" sync=FAIL-OPEN (%v)", dw.SyncErr)
			} else {
				fmt.Printf(" sync=ok")
			}
		}
		fmt.Println(" session_id=" + sessionID)
	}

	// 2) Async recall with session correlation
	if _, err := mesh.RequestMemoryRecallFull(ctx, iomeshclient.MemoryRecallRequest{
		TenantID:  tenant,
		Query:     "iomesh-client-sdk-go memory-metering-dogfood",
		Limit:     8,
		SessionID: sessionID,
	}); err != nil {
		log.Printf("WARN RequestMemoryRecallFull: %v", err)
	} else {
		fmt.Println("PASS RequestMemoryRecallFull session_id=" + sessionID)
	}

	// 3) Sync retrieve against memory sidecar (or gateway)
	res, err := memory.RetrieveMemory(ctx, iomeshclient.MemoryRetrieveRequest{
		TenantID:  tenant,
		Query:     "iomesh-client-sdk-go memory-metering-dogfood",
		SessionID: sessionID,
		Limit:     8,
	})
	if err != nil {
		log.Printf("WARN RetrieveMemory: %v (set IOMESH_MEMORY_ENDPOINT to sidecar if broker-only)", err)
	} else {
		fmt.Printf("PASS RetrieveMemory path=%s hits=%d\n", res.Path, len(res.Memories))
	}

	// 4) Remote metering emit (platform dashboards)
	if _, err := mesh.EmitLLMCall(ctx, iomeshclient.LLMCallEvent{
		Tenant:       tenant,
		SessionID:    sessionID,
		Model:        "sdk-dogfood",
		ModelID:      "sdk-dogfood",
		DurationMS:   1,
		Attempts:     1,
		TotalTokens:  0,
		PromptTokens: 0,
		Extra: map[string]any{
			"source": "iomesh-client-sdk-go",
			"probe":  "memory-metering-dogfood",
		},
	}); err != nil {
		log.Printf("WARN EmitLLMCall: %v", err)
	} else {
		fmt.Println("PASS EmitLLMCall type=dept.agent.llm_call")
	}

	fmt.Println("RESULT=done")
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
