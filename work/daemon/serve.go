// work/daemon/serve.go

package daemon

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

// ServeOptions configures Serve's shutdown.
type ServeOptions struct {
	// Timeout bounds the graceful shutdown. Zero means 15s.
	Timeout time.Duration

	// BeforeShutdown runs in order before the server drains. Close an sse.Broker
	// here, or Shutdown waits out the timeout on its streams.
	BeforeShutdown []func()

	// AfterServe runs in order after the server drains, such as closing the store.
	AfterServe []func() error
}

// Serve runs h on ln until ctx ends or it fails, then BeforeShutdown, drain and AfterServe; it returns the first error.
func Serve(ctx context.Context, ln net.Listener, h http.Handler, opts ServeOptions) error {
	srv := &http.Server{Handler: h, ReadHeaderTimeout: 10 * time.Second}

	serveErr := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	var retErr error
	select {
	case retErr = <-serveErr:
	case <-ctx.Done():
	}

	// End streams first, so Shutdown does not wait out the timeout on them.
	for _, fn := range opts.BeforeShutdown {
		fn()
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	sctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil && retErr == nil {
		retErr = err
	}

	// Release resources after the server has stopped accepting and draining.
	for _, fn := range opts.AfterServe {
		if err := fn(); err != nil && retErr == nil {
			retErr = err
		}
	}
	return retErr
}
