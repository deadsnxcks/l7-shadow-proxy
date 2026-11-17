//go:build unix

package reloader

import (
	"os"
	"syscall"
	"testing"
)

func TestReloader_SIGHUP(t *testing.T) {
	path, calls := startReloader(t, nil)
	base := warmUp(t, path, calls) // после разогрева signal.Notify уже точно вызван

	if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatalf("send SIGHUP: %v", err)
	}

	waitFor(t, calls, base+1)
}
