package fs

import (
	"fmt"
	"syscall"
)

// MountDevTmpfs mounts a tmpfs filesystem at /dev inside the container.
// This provides an empty, writable /dev directory for device nodes.
// Must be called after pivot_root when "/" is the container's root.
func MountDevTmpfs() error {
	// Mount tmpfs at /dev with mode 755 (rwxr-xr-x)
	// - "tmpfs" as source is conventional (not a real path)
	// - "mode=755" sets directory permissions
	if err := syscall.Mount(
		"tmpfs",
		"/dev",
		"tmpfs",
		0,
		"mode=755"); err != nil {
		return fmt.Errorf("mount tmpfs on /dev: %w", err)
	}
	return nil
}

// CreateDeviceNodes creates essential device nodes in /dev.
// Devices: null, zero, random, urandom, tty
// Must be called after MountDevTmpfs().
func CreateDeviceNodes() error {
	panic("TODO: not implemented")
}

// createDeviceNode creates a single device node at the given path.
// Parameters:
//   - path: absolute path (e.g., "/dev/null")
//   - mode: file type and permissions (e.g., unix.S_IFCHR | 0666)
//   - major: device major number
//   - minor: device minor number
func createDeviceNode(path string, mode uint32, major uint32, minor uint32) error {
	panic("TODO: not implemented")
}
