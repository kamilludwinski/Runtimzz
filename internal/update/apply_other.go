//go:build !windows

package update

import (
	"os"
	"syscall"
	"time"
)

func waitForProcessExit(pid int) error {
	// On Unix we can wait for the process; if we don't have permission, poll until it's gone.
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	// Wait for the process to exit (works when we're the child of pid, or on many systems for any process).
	state, err := proc.Wait()
	if err == nil {
		_ = state
		return nil
	}
	// Fallback: poll until process disappears
	for i := 0; i < 60; i++ {
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}
