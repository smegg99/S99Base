// work/applife/applife.go

// Package applife shuts an app down in reverse order under one deadline.
package applife

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Closer is anything that can be shut down.
type Closer interface{ Close() error }

// CloserFunc adapts a plain function.
type CloserFunc func() error

func (f CloserFunc) Close() error { return f() }

// contextCloser is satisfied by WithContext, so the shutdown deadline reaches it.
type contextCloser interface {
	closeContext(ctx context.Context) error
}

type ctxCloser struct{ fn func(context.Context) error }

func (c *ctxCloser) Close() error { return c.fn(context.Background()) }

func (c *ctxCloser) closeContext(ctx context.Context) error { return c.fn(ctx) }

// WithContext adapts a Shutdown(ctx) function such as http.Server's, with the deadline from Options.Timeout.
func WithContext(fn func(context.Context) error) Closer { return &ctxCloser{fn: fn} }

type Options struct {
	// Timeout is shared among remaining closers; each gets a bounded turn. Zero means 15s.
	Timeout time.Duration
	// Closers in construction order. Shutdown reverses them.
	Closers []Closer
}

// Lifecycle owns one app's shutdown.
type Lifecycle struct {
	opts Options
	once sync.Once
	err  error
}

func New(opts Options) *Lifecycle {
	if opts.Timeout <= 0 {
		opts.Timeout = 15 * time.Second
	}
	return &Lifecycle{opts: opts}
}

// Wait blocks until SIGINT, SIGTERM or ctx; after a signal a second one exits with 130.
func (l *Lifecycle) Wait(ctx context.Context) {
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	select {
	case <-ch:
		go func() {
			<-ch
			os.Exit(130)
		}()
	case <-ctx.Done():
		signal.Stop(ch)
	}
}

// Shutdown closes everything in reverse construction order; later calls return the first error.
func (l *Lifecycle) Shutdown() error {
	l.once.Do(func() { l.err = l.shutdown() })
	return l.err
}

func (l *Lifecycle) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), l.opts.Timeout)
	defer cancel()

	var errs []error
	for i := len(l.opts.Closers) - 1; i >= 0; i-- {
		deadline, _ := ctx.Deadline()
		turn, stop := context.WithTimeout(ctx, time.Until(deadline)/time.Duration(i+1))
		errs = append(errs, closeOne(turn, l.opts.Closers[i]))
		stop()
	}
	return errors.Join(errs...)
}

func closeOne(ctx context.Context, c Closer) error {
	done := make(chan error, 1)
	go func() {
		if cc, ok := c.(contextCloser); ok {
			done <- cc.closeContext(ctx)
		} else {
			done <- c.Close()
		}
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("applife: close did not finish: %w", ctx.Err())
	}
}
