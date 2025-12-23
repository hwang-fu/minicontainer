package container

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/hwang-fu/minicontainer/cmd"
	"github.com/hwang-fu/minicontainer/runtime"
)

// prepareRootfs creates necessary directories inside rootfs before namespace entry.
// This must be done in the parent process to avoid permission issues in user namespace.
func prepareRootfs(rootfsPath string) error {
	if rootfsPath == "" {
		return nil
	}
	// Create .pivot_root directory for pivot_root syscall
	if err := os.MkdirAll(filepath.Join(rootfsPath, ".pivot_root"), 0700); err != nil {
		return err
	}
	// Create mount points for virtual filesystems
	for _, dir := range []string{"proc", "sys"} {
		if err := os.MkdirAll(filepath.Join(rootfsPath, dir), 0755); err != nil {
			return err
		}
	}
	return nil
}

// BuildEnv creates environment variables to pass to init process.
func BuildEnv(cfg cmd.ContainerConfig) []string {
	env := os.Environ()
	if cfg.RootfsPath != "" {
		env = append(env, "MINICONTAINER_ROOTFS="+cfg.RootfsPath)
	}
	if cfg.Hostname != "" {
		env = append(env, "MINICONTAINER_HOSTNAME="+cfg.Hostname)
	}
	for _, e := range cfg.Env {
		env = append(env, "MINICONTAINER_ENV_"+e)
	}
	return env
}

// RunWithTTY runs the container with pseudo-terminal for interactive mode.
// Creates PTY, sets raw mode, and relays I/O between terminal and container.
func RunWithTTY(cfg cmd.ContainerConfig, cmdArgs []string) {
	// Create necessary directories before entering namespaces (avoids permission issues)
	if err := prepareRootfs(cfg.RootfsPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to prepare rootfs: %v\n", err)
		os.Exit(1)
	}

	master, slave, err := runtime.OpenPTY()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create pty: %v\n", err)
		os.Exit(1)
	}
	defer master.Close()
	defer slave.Close()

	// SetRawMode returns (restoreFunc, error) - restoreFunc resets terminal on exit
	restoreFunc, err := runtime.SetRawMode(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set raw mode: %v\n", err)
		os.Exit(1)
	}
	defer restoreFunc()

	cmd := exec.Command("/proc/self/exe", append([]string{"init"}, cmdArgs...)...)
	cmd.Stdin = slave
	cmd.Stdout = slave
	cmd.Stderr = slave

	// Set up namespace flags
	// Skip CLONE_NEWUSER when running as root - it causes restrictions on mount operations
	cloneFlags := syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWIPC | syscall.CLONE_NEWNS
	sysProcAttr := &syscall.SysProcAttr{
		Cloneflags: uintptr(cloneFlags),
		Setsid:     true,
	}
	if os.Getuid() != 0 {
		// Running as non-root: use user namespace with UID/GID mappings
		sysProcAttr.Cloneflags |= syscall.CLONE_NEWUSER
		sysProcAttr.UidMappings = []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getuid(), Size: 1}}
		sysProcAttr.GidMappings = []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getgid(), Size: 1}}
	}
	cmd.SysProcAttr = sysProcAttr
	cmd.Env = append(BuildEnv(cfg), "MINICONTAINER_TTY=1")

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	slave.Close() // Close slave in parent after child starts

	// Relay I/O: terminal <-> PTY master
	go io.Copy(master, os.Stdin)
	go io.Copy(os.Stdout, master)

	cmd.Wait()
	master.Close() // Stops io.Copy goroutines
	restoreFunc()  // Restore terminal - must call explicitly before exit
}

// RunWithoutTTY runs container with direct stdin/stdout passthrough.
func RunWithoutTTY(cfg cmd.ContainerConfig, cmdArgs []string) {
	// Create necessary directories before entering namespaces (avoids permission issues)
	if err := prepareRootfs(cfg.RootfsPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to prepare rootfs: %v\n", err)
		os.Exit(1)
	}

	cmd := exec.Command("/proc/self/exe", append([]string{"init"}, cmdArgs...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up namespace flags
	// Skip CLONE_NEWUSER when running as root - it causes restrictions on mount operations
	cloneFlags := syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWIPC | syscall.CLONE_NEWNS
	sysProcAttr := &syscall.SysProcAttr{
		Cloneflags: uintptr(cloneFlags),
	}
	if os.Getuid() != 0 {
		// Running as non-root: use user namespace with UID/GID mappings
		sysProcAttr.Cloneflags |= syscall.CLONE_NEWUSER
		sysProcAttr.UidMappings = []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getuid(), Size: 1}}
		sysProcAttr.GidMappings = []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getgid(), Size: 1}}
	}
	cmd.SysProcAttr = sysProcAttr
	cmd.Env = BuildEnv(cfg)

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
