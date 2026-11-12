// common/ttlcache/ttlcache.go

// Package ttlcache holds one value behind a TTL; concurrent misses share one fetch.
package ttlcache

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Cached is one value refreshed by fetch when stale. Use New; the zero value is unusable.
type Cached[T any] struct {
	ttl            time.Duration
	refreshTimeout time.Duration
	fetch          func(context.Context) (T, error)

	group singleflight.Group

	mu     sync.Mutex
	value  T
	at     time.Time
	loaded bool
}

// New caches for ttl and bounds shared refreshes to 30 seconds; zero ttl refreshes every Get.
func New[T any](ttl time.Duration, fetch func(context.Context) (T, error)) *Cached[T] {
	return &Cached[T]{ttl: ttl, fetch: fetch, refreshTimeout: 30 * time.Second}
}

// Get returns the value, refreshing it when stale; a failed refresh returns the last good value without an error.
func (c *Cached[T]) Get(ctx context.Context) (T, error) {
	if v, ok := c.fresh(); ok {
		return v, nil
	}

	if err := ctx.Err(); err != nil {
		var zero T
		return zero, err
	}
	wait, cancel := context.WithTimeout(ctx, c.refreshTimeout)
	defer cancel()
	result := c.group.DoChan("", func() (any, error) {
		if v, ok := c.fresh(); ok {
			return &v, nil
		}
		shared, stop := context.WithTimeout(context.WithoutCancel(ctx), c.refreshTimeout)
		defer stop()
		v, err := c.fetch(shared)
		if err == nil {
			err = shared.Err()
		}
		if err != nil {
			return nil, err
		}
		c.put(v)
		return &v, nil
	})
	select {
	case <-wait.Done():
		if err := ctx.Err(); err != nil {
			var zero T
			return zero, err
		}
		return c.failed(wait.Err())
	case result := <-result:
		if result.Err != nil {
			return c.failed(result.Err)
		}
		return *result.Val.(*T), nil
	}
}

func (c *Cached[T]) failed(err error) (T, error) {
	if stale, ok := c.last(); ok {
		return stale, nil
	}
	var zero T
	return zero, err
}

func (c *Cached[T]) fresh() (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.loaded || time.Since(c.at) >= c.ttl {
		var zero T
		return zero, false
	}
	return c.value, true
}

func (c *Cached[T]) last() (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.loaded {
		var zero T
		return zero, false
	}
	return c.value, true
}

func (c *Cached[T]) put(v T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value = v
	c.at = time.Now()
	c.loaded = true
}
