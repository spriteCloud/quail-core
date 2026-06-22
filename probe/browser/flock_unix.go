//go:build !windows

package browser

import (
	"os"
	"syscall"
)

// lockRunner takes an exclusive advisory lock on the runner-cache lock
// file so two concurrent quail probes don't race on `npm install`.
func lockRunner(f *os.File) error { return syscall.Flock(int(f.Fd()), syscall.LOCK_EX) }

func unlockRunner(f *os.File) error { return syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }
