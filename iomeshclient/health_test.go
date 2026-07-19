package iomeshclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
)

func TestHealthOK(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if err := nc.Health(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(gotUA, "iomesh-client-sdk-go/") {
		t.Fatalf("User-Agent=%q", gotUA)
	}
}

func TestHealthNonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	err = nc.Health(context.Background())
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("err=%v", err)
	}
}

func TestReadyOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ready" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if err := nc.Ready(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestReadyFallbackReadyz(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/readyz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if err := nc.Ready(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(paths) < 2 || paths[0] != "/ready" || paths[1] != "/readyz" {
		t.Fatalf("paths=%v", paths)
	}
}

func TestReadyBothMissing(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	err = nc.Ready(context.Background())
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("err=%v", err)
	}
}
