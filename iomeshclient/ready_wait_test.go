package iomeshclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
)

func TestWaitReady_SucceedsAfterNFailures(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ready" {
			http.NotFound(w, r)
			return
		}
		if n.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := nc.WaitReady(ctx, iomeshclient.WaitReadyOptions{Interval: 20 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	if got := n.Load(); got < 3 {
		t.Fatalf("attempts=%d", got)
	}
}

func TestWaitReady_TimeoutFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	err = nc.WaitReady(ctx, iomeshclient.WaitReadyOptions{Interval: 15 * time.Millisecond})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "deadline") && !strings.Contains(err.Error(), "canceled") {
		// context.DeadlineExceeded message typically includes "deadline exceeded"
		if ctx.Err() == nil {
			t.Fatalf("err=%v", err)
		}
	}
}

func TestWaitReady_RequireHealth(t *testing.T) {
	var readyOK, healthOK atomic.Bool
	// Flip ready first, then health on a later attempt.
	var readyHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ready":
			readyHits.Add(1)
			if readyOK.Load() {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusServiceUnavailable)
		case "/health":
			if healthOK.Load() {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}

	// Start with neither ready.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- nc.WaitReady(ctx, iomeshclient.WaitReadyOptions{
			Interval:      15 * time.Millisecond,
			RequireHealth: true,
		})
	}()

	// After a few probes, allow ready only — still blocked on health.
	time.Sleep(50 * time.Millisecond)
	readyOK.Store(true)
	time.Sleep(50 * time.Millisecond)
	// Should not have completed yet (health still failing).
	select {
	case err := <-done:
		t.Fatalf("completed early: %v", err)
	default:
	}
	healthOK.Store(true)

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for WaitReady")
	}
	if readyHits.Load() < 1 {
		t.Fatal("expected ready probes")
	}
}

func TestWaitReady_NilClient(t *testing.T) {
	var c *iomeshclient.Client
	err := c.WaitReady(context.Background(), iomeshclient.WaitReadyOptions{})
	if err == nil || !strings.Contains(err.Error(), "nil client") {
		t.Fatalf("err=%v", err)
	}
}

func TestWaitReadyElapsed_SucceedsAfterNFailures(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ready" {
			http.NotFound(w, r)
			return
		}
		if n.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	elapsed, err := nc.WaitReadyElapsed(ctx, iomeshclient.WaitReadyOptions{Interval: 20 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if elapsed < 0 {
		t.Fatalf("elapsed=%v want >= 0", elapsed)
	}
	// Two failed probes + interval sleeps before success → typically >0.
	if elapsed == 0 {
		t.Fatalf("elapsed=%v want > 0 after delayed success", elapsed)
	}
	if got := n.Load(); got < 3 {
		t.Fatalf("attempts=%d", got)
	}
}

func TestWaitReadyElapsed_TimeoutFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	nc, err := iomeshclient.Connect(iomeshclient.Options{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	elapsed, err := nc.WaitReadyElapsed(ctx, iomeshclient.WaitReadyOptions{Interval: 15 * time.Millisecond})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed <= 0 {
		t.Fatalf("elapsed=%v want > 0 on timeout", elapsed)
	}
	if !strings.Contains(err.Error(), "deadline") && !strings.Contains(err.Error(), "canceled") {
		if ctx.Err() == nil {
			t.Fatalf("err=%v", err)
		}
	}
}

func TestWaitReadyElapsed_NilClient(t *testing.T) {
	var c *iomeshclient.Client
	elapsed, err := c.WaitReadyElapsed(context.Background(), iomeshclient.WaitReadyOptions{})
	if err == nil || !strings.Contains(err.Error(), "nil client") {
		t.Fatalf("err=%v", err)
	}
	if elapsed != 0 {
		t.Fatalf("elapsed=%v want 0", elapsed)
	}
}
