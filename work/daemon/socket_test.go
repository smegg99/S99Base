// work/daemon/socket_test.go

package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestXDGSocketUsesRuntimeDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	path, ok := XDGSocket("myapp", "app.sock")
	if !ok {
		t.Fatal("ok = false, want true with XDG_RUNTIME_DIR set")
	}
	if want := filepath.Join(dir, "myapp", "app.sock"); path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	// The parent directory must exist afterwards, ready for Listen.
	if fi, err := os.Stat(filepath.Join(dir, "myapp")); err != nil || !fi.IsDir() {
		t.Errorf("parent dir not created (err=%v)", err)
	}
}

func TestXDGSocketFalseWhenUnset(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")
	if _, ok := XDGSocket("myapp", "app.sock"); ok {
		t.Error("ok = true, want false when XDG_RUNTIME_DIR is unset")
	}
}

func TestXDGSocketFalseWhenPathTooLong(t *testing.T) {
	deep := "/" + strings.Repeat("x", afUnixBound)
	t.Setenv("XDG_RUNTIME_DIR", deep)
	if _, ok := XDGSocket("app", "s.sock"); ok {
		t.Error("ok = true, want false when the path exceeds the AF_UNIX bound")
	}
}
