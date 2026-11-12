// work/daemon/daemon.go

// Package daemon runs HTTP over a unix socket and starts the daemon on demand.
package daemon

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var (
	// ErrRunning is returned by Listen when a live process already answers on the socket.
	ErrRunning = errors.New("daemon: already running on this socket")
	// ErrNotSocket is returned by Listen when the path holds something else, which it never removes.
	ErrNotSocket = errors.New("daemon: path exists and is not a socket")
)

// dialTimeout bounds the liveness probe against a socket whose owner has died.
const dialTimeout = 500 * time.Millisecond

// Transport returns an http.Transport that dials the unix socket at path for every host.
func Transport(path string) *http.Transport {
	return &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", path)
		},
	}
}

// Dial returns an http.Client for the unix socket at path; the URL host is ignored.
func Dial(path string) *http.Client {
	return &http.Client{Transport: Transport(path)}
}

// Listen takes the socket at path (ErrRunning if live, stale removed); a new dir is 0700, an existing one is kept, so keep it private: the socket is chmodded 0600 only after bind.
func Listen(path string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}

	// A socket that exists but does not answer is left over from a crash.
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("%w: %s", ErrNotSocket, path)
		}
		if live(path) {
			return nil, ErrRunning
		}
		if err := os.Remove(path); err != nil {
			return nil, err
		}
	}

	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		_ = ln.Close()
		return nil, err
	}
	return ln, nil
}

// live reports whether something is accepting connections on the socket.
func live(path string) bool {
	conn, err := net.DialTimeout("unix", path, dialTimeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
