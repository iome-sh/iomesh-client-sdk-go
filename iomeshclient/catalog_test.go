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

func TestListCatalog_ProductsAndFailOpen(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/catalog/data-products" {
			http.NotFound(w, r)
			return
		}
		gotUA = r.Header.Get("User-Agent")
		if r.URL.Query().Get("tenant") != "acme" {
			t.Errorf("tenant=%q", r.URL.Query().Get("tenant"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"products": []map[string]string{
				{"id": "ops-incidents", "layer": "operational", "subject": "dept.sre.incidents", "title": "Incidents"},
				{"id": "crm-contacts", "layer": "knowledge", "subject": "dept.sales.contacts", "name": "CRM"},
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
	res := c.ListCatalog(context.Background(), "")
	if res.Source != "mesh" || len(res.Products) != 2 {
		t.Fatalf("%+v", res)
	}
	if !strings.HasPrefix(gotUA, "iomesh-client-sdk-go/") {
		t.Fatalf("User-Agent=%q", gotUA)
	}
	out := iomeshclient.FormatCatalog(res)
	if !strings.Contains(out, "ops-incidents") || !strings.Contains(out, "operational") {
		t.Fatal(out)
	}

	empty := httptest.NewServer(http.NotFoundHandler())
	defer empty.Close()
	c2, err := iomeshclient.Connect(iomeshclient.Options{URL: empty.URL})
	if err != nil {
		t.Fatal(err)
	}
	res2 := c2.ListCatalog(context.Background(), "q")
	if res2.Source != "fail-open" {
		t.Fatalf("%+v", res2)
	}
}

func TestListCatalog_PortalFederation(t *testing.T) {
	// Broker paths 404; portal v17 succeeds with portal field names.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v17/portal/catalog/data-products":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"version": "v17-test",
				"products": []map[string]any{
					{
						"id":              "engineering-github-events",
						"name":            "GitHub Events",
						"mesh_layer":      "operational",
						"subject_pattern": "dept.engineering.github.>",
						"summary":         "GitHub webhook stream",
						"sample_subjects": []string{"dept.engineering.github.push"},
						"lineage":         []string{"github", "connector", "mesh"},
						"status":          "ga",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	res := c.ListCatalog(context.Background(), "")
	if res.Source != "portal" || !strings.Contains(res.Detail, "/v17/") {
		t.Fatalf("%+v", res)
	}
	if len(res.Products) != 1 {
		t.Fatalf("%+v", res)
	}
	p := res.Products[0]
	if p.Layer != "operational" || p.Subject == "" || p.Description == "" {
		t.Fatalf("normalize failed: %+v", p)
	}
	out := iomeshclient.FormatCatalog(res)
	if !strings.Contains(out, "engineering-github") || !strings.Contains(out, "source=portal") {
		t.Fatal(out)
	}
}

func TestGetCatalogProduct_Detail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v17/portal/catalog/data-products/engineering-github-events" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "engineering-github-events", "name": "GitHub Events",
				"mesh_layer": "operational", "summary": "detail ok",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	c, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	p, meta := c.GetCatalogProduct(context.Background(), "engineering-github-events")
	if meta.Source != "portal" || p.ID != "engineering-github-events" {
		t.Fatalf("p=%+v meta=%+v", p, meta)
	}
	d := iomeshclient.FormatProductDetail(p, meta)
	if !strings.Contains(d, "detail ok") {
		t.Fatal(d)
	}
}

func TestListCatalog_NilClient(t *testing.T) {
	var c *iomeshclient.Client
	res := c.ListCatalog(context.Background(), "")
	if res.Source != "off" || !strings.Contains(res.Detail, "nil client") {
		t.Fatalf("%+v", res)
	}
	_, meta := c.GetCatalogProduct(context.Background(), "x")
	if meta.Source != "off" {
		t.Fatalf("%+v", meta)
	}
}

func TestDecodeCatalogArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/catalog/data-products" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == "/v1/catalog/products" {
			_ = json.NewEncoder(w).Encode([]map[string]string{{"id": "x", "layer": "analytical"}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	c, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	res := c.ListCatalog(context.Background(), "")
	if res.Source != "mesh" || len(res.Products) != 1 || res.Products[0].ID != "x" {
		t.Fatalf("%+v", res)
	}
}

func TestListCatalog_LayerQuery(t *testing.T) {
	var gotQ, gotLayer string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/catalog/data-products" {
			http.NotFound(w, r)
			return
		}
		gotQ = r.URL.Query().Get("q")
		gotLayer = r.URL.Query().Get("mesh_layer")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"products": []map[string]string{{"id": "ops", "layer": "operational"}},
		})
	}))
	defer srv.Close()
	c, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	res := c.ListCatalog(context.Background(), "operational")
	if res.Source != "mesh" || gotQ != "operational" || gotLayer != "operational" {
		t.Fatalf("res=%+v q=%q mesh_layer=%q", res, gotQ, gotLayer)
	}
}
