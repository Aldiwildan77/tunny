//go:build linux || darwin

package daemon

import (
	"errors"
	"syscall"
)

func IsRunning(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func Terminate(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}
