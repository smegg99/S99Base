// work/daemon/spawn_unix.go

//go:build unix

package daemon

import (
	"os/exec"
	"syscall"
)

// spawn starts the daemon in its own session, so signals to the CLI's terminal leave it running.
func spawn(command []string) error {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil

	if err := cmd.Start(); err != nil {
		return err
	}
	// The child is never waited on.
	return cmd.Process.Release()
}
