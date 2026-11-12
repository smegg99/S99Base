// work/daemon/serve_test.go

package daemon

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

// cancelling ctx stops Serve and still runs AfterServe.
func TestServeReturnsOnContextCancel(t *testing.T) {
	sock := tempSocket(t)
	ln, err := Listen(sock)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	var afterServeRan atomic.Bool
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, ln, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}), ServeOptions{
			AfterServe: []func() error{func() error {
				afterServeRan.Store(true)
				return nil
			}},
		})
	}()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve err = %v, want nil on context cancel", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return after ctx cancel")
	}
	if !afterServeRan.Load() {
		t.Fatal("AfterServe did not run")
	}
}

// An open SSE stream must not make Shutdown wait out the timeout.
func TestServeClosesStreamsBeforeDrain(t *testing.T) {
	sock := tempSocket(t)
	ln, err := Listen(sock)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	stream, stop := testStream(t)
	mux := http.NewServeMux()
	mux.Handle("/events", stream)

	ctx, cancel := context.WithCancel(context.Background())
	const timeout = 2 * time.Second

	// Serve must be accepting before the client opens a stream.
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, ln, mux, ServeOptions{
			Timeout:        timeout,
			BeforeShutdown: []func(){stop},
		})
	}()

	client := Dial(sock)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://daemon/events", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	defer resp.Body.Close()

	start := time.Now()
	cancel()

	select {
	case err := <-done:
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("Serve err = %v, want nil", err)
		}
		if elapsed >= time.Second {
			t.Fatalf("Serve took %s, want well under the %s timeout", elapsed, timeout)
		}
		t.Logf("Serve returned in %s with BeforeShutdown closing the stream (timeout was %s)", elapsed, timeout)
	case <-time.After(3 * time.Second):
		t.Fatal("Serve did not return")
	}
}

// Control for TestServeClosesStreamsBeforeDrain.
func TestServeWithoutBeforeShutdownBlocksForTimeout(t *testing.T) {
	sock := tempSocket(t)
	ln, err := Listen(sock)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	stream, _ := testStream(t)
	mux := http.NewServeMux()
	mux.Handle("/events", stream)

	ctx, cancel := context.WithCancel(context.Background())
	const timeout = 700 * time.Millisecond

	// Serve must be accepting before the client opens a stream.
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, ln, mux, ServeOptions{Timeout: timeout})
	}()

	client := Dial(sock)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://daemon/events", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	defer resp.Body.Close()

	start := time.Now()
	cancel()

	select {
	case err := <-done:
		elapsed := time.Since(start)
		if err == nil {
			t.Fatal("Serve err = nil, want the deadline error from a Shutdown that could not drain the open stream")
		}
		if elapsed < timeout {
			t.Fatalf("Serve took %s, want at least the %s timeout - an open stream should have blocked Shutdown", elapsed, timeout)
		}
		t.Logf("without BeforeShutdown, Serve took %s to hit the %s timeout", elapsed, timeout)
	case <-time.After(3 * time.Second):
		t.Fatal("Serve did not return")
	}
}

// A listener that fails to accept must still run AfterServe.
func TestServeReturnsServeError(t *testing.T) {
	sock := tempSocket(t)
	ln, err := Listen(sock)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	// Accept then fails with a plain error, not http.ErrServerClosed.
	if err := ln.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var afterServeRan atomic.Bool
	err = Serve(context.Background(), ln, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), ServeOptions{
		AfterServe: []func() error{func() error {
			afterServeRan.Store(true)
			return nil
		}},
	})
	if err == nil {
		t.Fatal("Serve err = nil, want the accept failure from a closed listener")
	}
	if !afterServeRan.Load() {
		t.Fatal("AfterServe did not run after a serve error")
	}
}
