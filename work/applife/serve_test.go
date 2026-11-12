// work/applife/serve_test.go

package applife_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/smegg99/s99base/work/applife"
)

// fakeServer is a server whose ListenAndServe returns whatever the test sends.
type fakeServer struct{ result chan error }

func newFakeServer() *fakeServer { return &fakeServer{result: make(chan error, 1)} }

func (f *fakeServer) ListenAndServe() error { return <-f.result }

func TestABindFailureEndsServeWithoutASignal(t *testing.T) {
	srv := newFakeServer()
	boom := errors.New("listen tcp 127.0.0.1:9250: address already in use")
	srv.result <- boom

	lc := applife.New(applife.Options{})

	done := make(chan error, 1)
	go func() { done <- lc.Serve(context.Background(), srv) }()

	select {
	case err := <-done:
		if !errors.Is(err, boom) {
			t.Fatalf("err = %v, want the bind failure", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve is still waiting for a signal that is never coming")
	}
}

func TestAClosedServerIsNotAFailure(t *testing.T) {
	srv := newFakeServer()
	srv.result <- http.ErrServerClosed

	if err := applife.New(applife.Options{}).Serve(context.Background(), srv); err != nil {
		t.Fatalf("err = %v, want nil for a server that was shut down", err)
	}
}

// Wait reports a signal by ending the context.
func TestServeReturnsWhenTheContextEnds(t *testing.T) {
	srv := newFakeServer() // never sends, so ListenAndServe blocks

	ctx, cancel := context.WithCancel(context.Background())
	lc := applife.New(applife.Options{})

	done := make(chan error, 1)
	go func() { done <- lc.Serve(ctx, srv) }()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("err = %v, want nil - stopping on a signal is not a failure", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return when the context ended")
	}

	// Let the serving goroutine finish, so the race detector sees it end.
	srv.result <- http.ErrServerClosed
}
