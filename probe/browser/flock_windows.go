//go:build windows

package browser

import "os"

// quail: no-op cache lock on Windows. The runner-cache lock is best-
// effort cross-process coordination; on Windows two concurrent probes
// might race on `npm install`, re-doing the same work — slowdown, not
// corruption. Upgrade path: golang.org/x/sys/windows LockFileEx.
func lockRunner(f *os.File) error { _ = f; return nil }

func unlockRunner(f *os.File) error { _ = f; return nil }
