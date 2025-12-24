package fs

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
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
	devices := []struct {
		// Device node specifications: path, mode, major, minor
		// All are character devices (S_IFCHR) with permissions 0666 (rw-rw-rw-)
		path  string
		mode  uint32
		major uint32
		minor uint32
	}{
		{"/dev/null", unix.S_IFCHR | 0o666, 1, 3},    // Data sink
		{"/dev/zero", unix.S_IFCHR | 0o666, 1, 5},    // Infinite zeros
		{"/dev/random", unix.S_IFCHR | 0o666, 1, 8},  // Blocking random
		{"/dev/urandom", unix.S_IFCHR | 0o666, 1, 9}, // Non-blocking random
		{"/dev/tty", unix.S_IFCHR | 0o666, 5, 0},     // Controlling terminal
	}

	for _, d := range devices {
		if err := createDeviceNode(d.path, d.mode, d.major, d.minor); err != nil {
			return err
		}
	}
	return nil
}

// createDeviceNode creates a single device node at the given path.
// Parameters:
//   - path: absolute path (e.g., "/dev/null")
//   - mode: file type and permissions (e.g., unix.S_IFCHR | 0666)
//   - major: device major number
//   - minor: device minor number
func createDeviceNode(path string, mode uint32, major uint32, minor uint32) error {
	// Mkdev encodes major/minor into the format expected by Mknod
	dev := unix.Mkdev(major, minor)

	// Mknod creates a special file (device node)
	if err := unix.Mknod(path, mode, int(dev)); err != nil {
		return fmt.Errorf("mknod %s: %w", path, err)
	}
	return nil
}
