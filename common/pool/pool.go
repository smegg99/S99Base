// common/pool/pool.go

// Package pool runs work items through bounded goroutines, keeping input order.
package pool

import (
	"context"
	"sync"
)

// Result pairs one item's output with its error.
type Result[R any] struct {
	Value R
	Err   error
}

// Run returns one Result per item in order; on cancellation the rest carry ctx.Err(), and workers is clamped to [1, len(items)].
func Run[T, R any](ctx context.Context, items []T, workers int, fn func(context.Context, T) (R, error)) []Result[R] {
	n := len(items)
	if n == 0 {
		return nil
	}
	if workers < 1 {
		workers = 1
	}
	if workers > n {
		workers = n
	}

	results := make([]Result[R], n)
	jobs := make(chan int)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				if ctx.Err() != nil {
					results[i] = Result[R]{Err: ctx.Err()}
					continue
				}
				v, err := fn(ctx, items[i])
				results[i] = Result[R]{Value: v, Err: err}
			}
		}()
	}

	for i := 0; i < n; i++ {
		select {
		case <-ctx.Done():
			for ; i < n; i++ {
				results[i] = Result[R]{Err: ctx.Err()}
			}
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()
	return results
}
