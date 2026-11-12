// work/daemon/spawn_other.go

//go:build !unix

package daemon

import "errors"

// spawn needs Setsid, so it is unix-only.
func spawn([]string) error {
	return errors.New("daemon: starting a detached daemon is only supported on unix")
}
