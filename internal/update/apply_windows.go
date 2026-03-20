//go:build windows

package update

import (
	"syscall"
)

const waitTimeoutMs = 120000 // 2 min

func waitForProcessExit(pid int) error {
	h, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(h)
	ev, err := syscall.WaitForSingleObject(h, waitTimeoutMs)
	if err != nil {
		return err
	}
	if ev != syscall.WAIT_OBJECT_0 {
		return err
	}
	return nil
}
