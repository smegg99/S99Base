// work/daemon/ensure.go

package daemon

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Config drives EnsureRunning.
type Config struct {
	// Socket is the unix socket the daemon listens on.
	Socket string

	// Command starts the daemon detached. Empty means EnsureRunning only checks.
	Command []string

	// Ready decides whether the daemon is serving. Nil means a connection is enough.
	Ready func(context.Context, *http.Client) error

	// Timeout bounds the wait for a started daemon. Zero means five seconds.
	Timeout time.Duration
}

// pollEvery is how often EnsureRunning re-checks a starting daemon.
const pollEvery = 50 * time.Millisecond

// EnsureRunning returns once a daemon answers on cfg.Socket, starting cfg.Command detached to outlive the caller.
func EnsureRunning(ctx context.Context, cfg Config) error {
	if cfg.Socket == "" {
		return errors.New("daemon: EnsureRunning needs a Socket")
	}

	// One client for every poll: a new transport per check would leak its idle connections.
	client := Dial(cfg.Socket)
	defer client.CloseIdleConnections()

	if check(ctx, cfg, client) == nil {
		return nil
	}
	if len(cfg.Command) == 0 {
		return fmt.Errorf("daemon: nothing answers on %s and no Command to start one", cfg.Socket)
	}
	if err := spawn(cfg.Command); err != nil {
		return fmt.Errorf("daemon: start %q: %w", cfg.Command[0], err)
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	ticker := time.NewTicker(pollEvery)
	defer ticker.Stop()
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	var last error
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("daemon: %s did not come up within %s: %w", cfg.Socket, timeout, last)
		case <-ticker.C:
			if last = check(ctx, cfg, client); last == nil {
				return nil
			}
		}
	}
}

// check reports whether the daemon is up and, when Ready is set, ready.
func check(ctx context.Context, cfg Config, client *http.Client) error {
	if !live(cfg.Socket) {
		return errNotUp
	}
	if cfg.Ready == nil {
		return nil
	}
	return cfg.Ready(ctx, client)
}

var errNotUp = errors.New("daemon: not up")
