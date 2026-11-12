// work/applife/applife_test.go

package applife_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/smegg99/s99base/work/applife"
)

func TestClosersRunInReverseOrder(t *testing.T) {
	var order []string

	err := applife.New(applife.Options{
		Closers: []applife.Closer{
			applife.CloserFunc(func() error { order = append(order, "db"); return nil }),
			applife.CloserFunc(func() error { order = append(order, "server"); return nil }),
		},
	}).Shutdown()
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if len(order) != 2 || order[0] != "server" || order[1] != "db" {
		t.Fatalf("order = %v, want [server db]", order)
	}
}

// Every closer runs and every error is returned.
func TestOneFailureDoesNotAbortTheRest(t *testing.T) {
	boom := errors.New("server: still draining")
	closed := false

	err := applife.New(applife.Options{
		Closers: []applife.Closer{
			applife.CloserFunc(func() error { closed = true; return nil }),
			applife.CloserFunc(func() error { return boom }),
		},
	}).Shutdown()
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want it to carry the failure", err)
	}
	if !closed {
		t.Fatal("a later closer was skipped after an earlier one failed")
	}
}

func TestShutdownIsIdempotent(t *testing.T) {
	calls := 0

	lc := applife.New(applife.Options{
		Closers: []applife.Closer{applife.CloserFunc(func() error { calls++; return nil })},
	})
	_ = lc.Shutdown()
	_ = lc.Shutdown()

	if calls != 1 {
		t.Fatalf("closer ran %d times, want 1", calls)
	}
}

func TestWithContextReceivesTheDeadline(t *testing.T) {
	var deadline time.Time
	var ok bool

	_ = applife.New(applife.Options{
		Timeout: 50 * time.Millisecond,
		Closers: []applife.Closer{
			applife.WithContext(func(ctx context.Context) error {
				deadline, ok = ctx.Deadline()
				return nil
			}),
		},
	}).Shutdown()

	if !ok {
		t.Fatal("ctx had no deadline")
	}
	if d := time.Until(deadline); d > 60*time.Millisecond {
		t.Fatalf("deadline is %v away, want the 50ms timeout", d)
	}
}

func TestWaitReturnsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() { applife.New(applife.Options{}).Wait(ctx); close(done) }()
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Wait ignored ctx cancellation")
	}
}

func TestSecondShutdownReportsTheFirstError(t *testing.T) {
	boom := errors.New("db: close failed")

	lc := applife.New(applife.Options{
		Closers: []applife.Closer{applife.CloserFunc(func() error { return boom })},
	})
	first := lc.Shutdown()
	second := lc.Shutdown()

	if !errors.Is(first, boom) {
		t.Fatalf("first = %v, want the failure", first)
	}
	if !errors.Is(second, boom) {
		t.Fatalf("second = %v, want the same failure reported again", second)
	}
}

// A closer that hangs must not hold the process past Timeout.
func TestTimeoutBoundsAPlainCloserThatHangs(t *testing.T) {
	hang := make(chan struct{})
	defer close(hang)
	l := applife.New(applife.Options{
		Timeout: 100 * time.Millisecond,
		Closers: []applife.Closer{applife.CloserFunc(func() error { <-hang; return nil })},
	})

	done := make(chan error, 1)
	go func() { done <- l.Shutdown() }()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Shutdown = %v, want the deadline", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown waited on a hung closer past its timeout")
	}
}

// A server holding a stream open uses the whole timeout; the logger and the store still close.
func TestClosersAfterTheDeadlineStillRun(t *testing.T) {
	hang := make(chan struct{})
	defer close(hang)
	var flushed bool
	l := applife.New(applife.Options{
		Timeout: 50 * time.Millisecond,
		Closers: []applife.Closer{
			applife.CloserFunc(func() error { flushed = true; return nil }),
			applife.CloserFunc(func() error { <-hang; return nil }),
		},
	})

	err := l.Shutdown()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown = %v, want the deadline from the hung closer", err)
	}
	if !flushed {
		t.Fatal("the closer after the deadline never ran")
	}
}

func TestContextClosersCannotExhaustTheCleanupBudget(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	flushed := make(chan struct{})
	closers := []applife.Closer{applife.CloserFunc(func() error { close(flushed); return nil })}
	for range 3 {
		closers = append(closers, applife.WithContext(func(context.Context) error { <-release; return nil }))
	}
	start := time.Now()
	err := applife.New(applife.Options{Timeout: 40 * time.Millisecond, Closers: closers}).Shutdown()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown = %v", err)
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Fatal("shutdown exceeded the total budget")
	}
	select {
	case <-flushed:
	default:
		t.Fatal("cleanup after blocked closers did not run")
	}
}
