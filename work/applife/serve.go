// work/applife/serve.go

package applife

import (
	"context"
	"errors"
	"net/http"
)

// Server is what Serve runs. *http.Server satisfies it.
type Server interface{ ListenAndServe() error }

// Serve runs srv until it stops or a signal arrives, so a failed bind exits; the caller shuts down.
func (l *Lifecycle) Serve(ctx context.Context, srv Server) error {
	// Buffered so the goroutine can report after Serve has returned.
	stopped := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		// ErrServerClosed means a clean Shutdown.
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		stopped <- err
	}()

	// Cancelled on return so Wait stops listening for signals.
	waiting, done := context.WithCancel(ctx)
	defer done()

	signalled := make(chan struct{})
	go func() {
		l.Wait(waiting)
		close(signalled)
	}()

	select {
	case err := <-stopped:
		return err
	case <-signalled:
		return nil
	}
}
