package iomeshclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
)

func TestQueryContext_TextAndLineage(t *testing.T) {
	var gotPath, gotUA, gotCT string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotUA = r.Header.Get("User-Agent")
		gotCT = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"text": "ops context for incidents",
			"lineage": []map[string]string{
				{"id": "ops-incidents", "subject": "dept.sre.incidents", "source": "mesh", "freshness": "1m"},
				{"product": "crm-contacts", "subject": "dept.sales.contacts"},
			},
		})
	}))
	defer srv.Close()

	c, err := iomeshclient.Connect(
		iomeshclient.Options{URL: srv.URL},
		iomeshclient.WithTenant("acme"),
	)
	if err != nil {
		t.Fatal(err)
	}
	res := c.QueryContext(context.Background(), iomeshclient.QueryContextRequest{
		Workspace:      "ws1",
		Query:          "incidents",
		Limit:          0, // default 20
		IncludeLineage: true,
	})
	if res.Text != "ops context for incidents" {
		t.Fatalf("text=%q", res.Text)
	}
	if len(res.Lineage) != 2 {
		t.Fatalf("lineage=%+v", res.Lineage)
	}
	if res.Lineage[0].ID != "ops-incidents" || res.Lineage[1].Product != "crm-contacts" {
		t.Fatalf("lineage fields: %+v", res.Lineage)
	}
	if gotPath != "/v1/context/query" {
		t.Fatalf("path=%q", gotPath)
	}
	if !strings.HasPrefix(gotUA, "iomesh-client-sdk-go/") {
		t.Fatalf("User-Agent=%q", gotUA)
	}
	if !strings.Contains(gotCT, "application/json") {
		t.Fatalf("Content-Type=%q", gotCT)
	}
	if gotBody["tenant"] != "acme" {
		t.Fatalf("tenant=%v", gotBody["tenant"])
	}
	if gotBody["workspace"] != "ws1" {
		t.Fatalf("workspace=%v", gotBody["workspace"])
	}
	if gotBody["query"] != "incidents" {
		t.Fatalf("query=%v", gotBody["query"])
	}
	if gotBody["limit"] != float64(20) {
		t.Fatalf("limit=%v (want default 20)", gotBody["limit"])
	}
	if gotBody["include_lineage"] != true {
		t.Fatalf("include_lineage=%v", gotBody["include_lineage"])
	}
}

func TestQueryContext_ItemsShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"text": "first chunk",
					"lineage": []map[string]string{
						{"id": "a", "subject": "s.a"},
					},
				},
				{
					"text": "second chunk",
					"lineage": []map[string]string{
						{"id": "b", "product": "prod-b"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	c, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL}, iomeshclient.WithTenant("t"))
	if err != nil {
		t.Fatal(err)
	}
	res := c.QueryContext(context.Background(), iomeshclient.QueryContextRequest{
		Query: "q",
	})
	if res.Text != "first chunk\nsecond chunk" {
		t.Fatalf("text=%q", res.Text)
	}
	if len(res.Lineage) != 2 || res.Lineage[0].ID != "a" || res.Lineage[1].ID != "b" {
		t.Fatalf("lineage=%+v", res.Lineage)
	}
}

func TestQueryContext_FailOpen(t *testing.T) {
	// Non-OK
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer bad.Close()
	c, err := iomeshclient.Connect(iomeshclient.Options{URL: bad.URL})
	if err != nil {
		t.Fatal(err)
	}
	res := c.QueryContext(context.Background(), iomeshclient.QueryContextRequest{Query: "x"})
	if res.Text != "" || len(res.Lineage) != 0 {
		t.Fatalf("want empty on non-OK: %+v", res)
	}

	// Transport error (closed server)
	closed := httptest.NewServer(http.NotFoundHandler())
	url := closed.URL
	closed.Close()
	c2, err := iomeshclient.Connect(iomeshclient.Options{URL: url})
	if err != nil {
		t.Fatal(err)
	}
	res2 := c2.QueryContext(context.Background(), iomeshclient.QueryContextRequest{Query: "x"})
	if res2.Text != "" || len(res2.Lineage) != 0 {
		t.Fatalf("want empty on transport: %+v", res2)
	}

	// Nil client
	var nilC *iomeshclient.Client
	res3 := nilC.QueryContext(context.Background(), iomeshclient.QueryContextRequest{Query: "x"})
	if res3.Text != "" || len(res3.Lineage) != 0 {
		t.Fatalf("want empty on nil: %+v", res3)
	}
}

func TestFormatContextSnippet(t *testing.T) {
	// Text only
	out := iomeshclient.FormatContextSnippet(iomeshclient.ContextResult{Text: "  hello mesh  "})
	if out != "hello mesh" {
		t.Fatalf("%q", out)
	}

	// Text + lineage block (max 12)
	refs := make([]iomeshclient.LineageRef, 14)
	for i := range refs {
		refs[i] = iomeshclient.LineageRef{
			ID:        "p" + strconv.Itoa(i),
			Subject:   "dept.x",
			Source:    "mesh",
			Freshness: "1m",
		}
	}
	out = iomeshclient.FormatContextSnippet(iomeshclient.ContextResult{
		Text:    "body",
		Lineage: refs,
	})
	if !strings.Contains(out, "body") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "<iomesh-lineage>") || !strings.Contains(out, "</iomesh-lineage>") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "subject=dept.x") || !strings.Contains(out, "source=mesh") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "…") {
		t.Fatalf("expected ellipsis for >12 refs: %s", out)
	}
	// Product fallback when ID empty
	out2 := iomeshclient.FormatContextSnippet(iomeshclient.ContextResult{
		Lineage: []iomeshclient.LineageRef{{Product: "only-product", Subject: "s1"}},
	})
	if !strings.Contains(out2, "only-product") || !strings.Contains(out2, "subject=s1") {
		t.Fatal(out2)
	}
}

func TestContextSnippet_Integration(t *testing.T) {
	var includeLineage any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		includeLineage = body["include_lineage"]
		_ = json.NewEncoder(w).Encode(map[string]any{
			"text": "snippet body",
			"lineage": []map[string]string{
				{"id": "dp1", "subject": "dept.ops", "source": "catalog"},
			},
		})
	}))
	defer srv.Close()

	c, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL}, iomeshclient.WithTenant("t"))
	if err != nil {
		t.Fatal(err)
	}
	snip := c.ContextSnippet(context.Background(), ".", "sdk dogfood")
	if !strings.Contains(snip, "snippet body") {
		t.Fatalf("snip=%q", snip)
	}
	if !strings.Contains(snip, "<iomesh-lineage>") || !strings.Contains(snip, "dp1") {
		t.Fatalf("snip lineage=%q", snip)
	}
	if includeLineage != true {
		t.Fatalf("ContextSnippet must set include_lineage=true, got %v", includeLineage)
	}

	// Fail-open empty
	var nilC *iomeshclient.Client
	if s := nilC.ContextSnippet(context.Background(), ".", "q"); s != "" {
		t.Fatalf("nil client snip=%q", s)
	}
}
