// common/ttlcache/ttlcache_test.go

package ttlcache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestServesCachedValueWithinTTL(t *testing.T) {
	var calls atomic.Int64
	c := New(time.Hour, func(context.Context) (int, error) {
		calls.Add(1)
		return 42, nil
	})

	for i := 0; i < 3; i++ {
		v, err := c.Get(context.Background())
		if err != nil || v != 42 {
			t.Fatalf("Get = %d, %v", v, err)
		}
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("fetched %d times, want 1", n)
	}
}

func TestRefreshesOnceStale(t *testing.T) {
	var calls atomic.Int64
	c := New(time.Millisecond, func(context.Context) (int64, error) {
		return calls.Add(1), nil
	})

	if v, _ := c.Get(context.Background()); v != 1 {
		t.Fatalf("first Get = %d, want 1", v)
	}
	time.Sleep(5 * time.Millisecond)
	if v, _ := c.Get(context.Background()); v != 2 {
		t.Errorf("Get after the TTL = %d, want a refreshed 2", v)
	}
}

func TestConcurrentMissesCollapseIntoOneFetch(t *testing.T) {
	var calls atomic.Int64
	release := make(chan struct{})
	c := New(time.Hour, func(context.Context) (int, error) {
		calls.Add(1)
		<-release // hold every caller inside the fetch
		return 7, nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if v, err := c.Get(context.Background()); err != nil || v != 7 {
				t.Errorf("Get = %d, %v", v, err)
			}
		}()
	}

	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()

	if n := calls.Load(); n != 1 {
		t.Errorf("fetched %d times for 50 concurrent readers, want 1", n)
	}
}

func TestFailedRefreshServesTheLastGoodValue(t *testing.T) {
	var fail atomic.Bool
	c := New(time.Millisecond, func(context.Context) (string, error) {
		if fail.Load() {
			return "", errors.New("upstream is down")
		}
		return "good", nil
	})

	if v, err := c.Get(context.Background()); err != nil || v != "good" {
		t.Fatalf("first Get = %q, %v", v, err)
	}

	fail.Store(true)
	time.Sleep(5 * time.Millisecond)

	v, err := c.Get(context.Background())
	if err != nil {
		t.Errorf("err = %v, want nil: a stale answer beats an outage", err)
	}
	if v != "good" {
		t.Errorf("Get = %q, want the last good value", v)
	}
}

func TestErrorsWhenThereIsNothingToServe(t *testing.T) {
	want := errors.New("upstream is down")
	c := New(time.Hour, func(context.Context) (string, error) { return "", want })

	v, err := c.Get(context.Background())
	if !errors.Is(err, want) {
		t.Errorf("err = %v, want %v", err, want)
	}
	if v != "" {
		t.Errorf("value = %q, want the zero value", v)
	}
}

func TestCallerCancellationDoesNotPoisonTheSharedFetch(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	c := New(time.Hour, func(ctx context.Context) (string, error) {
		close(started)
		<-release
		return "fetched", ctx.Err()
	})
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { _, err := c.Get(ctx); done <- err }()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel = %v", err)
	}
	close(release)
	if value, err := c.Get(t.Context()); value != "fetched" || err != nil {
		t.Fatalf("shared fetch = %q, %v", value, err)
	}
}

func TestRefreshHasItsOwnDeadline(t *testing.T) {
	c := New(time.Hour, func(ctx context.Context) (int, error) { <-ctx.Done(); return 0, ctx.Err() })
	c.refreshTimeout = 5 * time.Millisecond
	if _, err := c.Get(t.Context()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("refresh = %v", err)
	}
}

func TestNilInterfaceValueDoesNotPanic(t *testing.T) {
	c := New(time.Hour, func(context.Context) (any, error) { return nil, nil })
	if value, err := c.Get(t.Context()); value != nil || err != nil {
		t.Fatalf("nil value = %v, %v", value, err)
	}
}
