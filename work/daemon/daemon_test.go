// work/daemon/daemon_test.go

package daemon

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func tempSocket(t *testing.T) string {
	t.Helper()
	// t.TempDir() can exceed the ~104-byte unix socket path limit.
	dir, err := os.MkdirTemp("", "d")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "s.sock")
}

func serve(t *testing.T, sock string, h http.Handler) {
	t.Helper()
	ln, err := Listen(sock)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	srv := &http.Server{Handler: h}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
}

func ok200(ctx context.Context, c *http.Client) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://daemon/", nil)
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("not ok")
	}
	return nil
}

// a request over the socket reaches the handler and comes back
func TestDialListenRoundTrip(t *testing.T) {
	sock := tempSocket(t)
	serve(t, sock, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "pong")
	}))

	resp, err := Dial(sock).Get("http://daemon/ping")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "pong" {
		t.Errorf("body = %q, want pong", body)
	}
}

// a second Listen on a live socket is refused, so a daemon is single-instance
func TestListenRefusesLiveSocket(t *testing.T) {
	sock := tempSocket(t)
	serve(t, sock, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	_, err := Listen(sock)
	if !errors.Is(err, ErrRunning) {
		t.Fatalf("second Listen err = %v, want ErrRunning", err)
	}
}

// a stale socket file left by a crash is cleared, not treated as a live owner
func TestListenClearsStaleSocket(t *testing.T) {
	sock := tempSocket(t)
	if err := os.MkdirAll(filepath.Dir(sock), 0o700); err != nil {
		t.Fatal(err)
	}
	// A crash leaves the socket file behind with nothing accepting on it.
	dead, err := net.ListenUnix("unix", &net.UnixAddr{Name: sock, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	dead.SetUnlinkOnClose(false)
	_ = dead.Close()

	ln, err := Listen(sock)
	if err != nil {
		t.Fatalf("Listen over stale socket: %v", err)
	}
	_ = ln.Close()
}

// A mistyped path must never cost the user a file.
func TestListenLeavesARegularFileAlone(t *testing.T) {
	sock := tempSocket(t)
	if err := os.MkdirAll(filepath.Dir(sock), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sock, []byte("notes"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Listen(sock); !errors.Is(err, ErrNotSocket) {
		t.Fatalf("Listen = %v, want ErrNotSocket", err)
	}
	if data, err := os.ReadFile(sock); err != nil || string(data) != "notes" {
		t.Fatalf("the file was touched: %q, %v", data, err)
	}
}

// EnsureRunning does not start anything when the daemon is already up
func TestEnsureRunningNoSpawnWhenUp(t *testing.T) {
	sock := tempSocket(t)
	serve(t, sock, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// A Command that would fail if it ran proves it does not run.
	err := EnsureRunning(ctx, Config{
		Socket:  sock,
		Command: []string{"/definitely/not/a/real/binary"},
		Ready:   ok200,
	})
	if err != nil {
		t.Fatalf("EnsureRunning on a live daemon = %v, want nil", err)
	}
}

// with nothing up and no command, EnsureRunning reports it rather than hanging
func TestEnsureRunningNoCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := EnsureRunning(ctx, Config{Socket: tempSocket(t)})
	if err == nil {
		t.Fatal("EnsureRunning with no command and nothing up = nil, want an error")
	}
}

// EnsureRunning starts a detached daemon and waits for it to answer
func TestEnsureRunningSpawns(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("detached spawn is unix-only")
	}
	sock := tempSocket(t)
	t.Setenv(helperSocketEnv, sock)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	err := EnsureRunning(ctx, Config{
		Socket:  sock,
		Command: []string{os.Args[0], "-test.run=TestHelperDaemon"},
		Ready:   ok200,
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("EnsureRunning spawn: %v", err)
	}
	if err := ok200(ctx, Dial(sock)); err != nil {
		t.Fatalf("daemon not answering after spawn: %v", err)
	}
}

const helperSocketEnv = "DAEMON_TEST_SOCKET"

// TestHelperDaemon is the process TestEnsureRunningSpawns starts; without its env var it does nothing.
func TestHelperDaemon(t *testing.T) {
	sock := os.Getenv(helperSocketEnv)
	if sock == "" {
		return
	}
	ln, err := Listen(sock)
	if err != nil {
		os.Exit(1)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})}
	go func() { _ = srv.Serve(ln) }()
	time.Sleep(3 * time.Second)
	_ = srv.Close()
}
