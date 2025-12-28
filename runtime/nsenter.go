package runtime

import (
	"fmt"
	"os"
	"runtime"

	"golang.org/x/sys/unix"
)

// NamespaceType represents a Linux namespace type.
type NamespaceType struct {
	Name string // e.g., "pid", "mnt", "net"
	Flag int    // e.g., unix.CLONE_NEWPID
}

// Namespaces to enter when exec'ing into a container.
// Order matters: some namespaces depend on others.
var execNamespaces = []NamespaceType{
	{"ipc", unix.CLONE_NEWIPC},
	{"uts", unix.CLONE_NEWUTS},
	{"net", unix.CLONE_NEWNET},
	{"pid", unix.CLONE_NEWPID},
	{"mnt", unix.CLONE_NEWNS},
}

// EnterNamespaces enters the namespaces of the given PID.
// Must be called before forking the exec process.
// Note: Entering PID namespace only affects children, not the current process.
func EnterNamespaces(pid int) error {
	// Lock OS thread - setns must happen on a single OS thread
	runtime.LockOSThread()

	for _, ns := range execNamespaces {
		path := fmt.Sprintf("/proc/%d/ns/%s", pid, ns.Name)

		fd, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open namespace %s: %w", ns.Name, err)
		}

		err = unix.Setns(int(fd.Fd()), ns.Flag)
		fd.Close()

		if err != nil {
			return fmt.Errorf("setns %s: %w", ns.Name, err)
		}
	}

	return nil
}
