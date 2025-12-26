package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

// RunInit is the container init process that runs INSIDE namespaces.
// It sets up the container environment and execs the user command.
// This is the "re-exec" pattern used by Docker/runc.
func RunInit(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "init requires a command")
		os.Exit(1)
	}

	// Set container hostname
	hostname := os.Getenv("MINICONTAINER_HOSTNAME")
	if hostname == "" {
		hostname = "minicontainer"
	}
	if err := syscall.Sethostname([]byte(hostname)); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set hostname: %v\n", err)
		os.Exit(1)
	}

	// Make all mounts private to this namespace
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		fmt.Fprintf(os.Stderr, "make mounts private failed: %v\n", err)
		os.Exit(1)
	}

	// Setup rootfs if specified
	rootfsPath := os.Getenv("MINICONTAINER_ROOTFS")
	if rootfsPath != "" {
		if err := setupRootfs(rootfsPath); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	}

	// Find and exec the command
	path, err := exec.LookPath(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "command not found: %s\n", args[0])
		os.Exit(1)
	}

	env := buildContainerEnv()
	if err := syscall.Exec(path, args, env); err != nil {
		fmt.Fprintf(os.Stderr, "exec failed: %v\n", err)
		os.Exit(1)
	}
}

// setupRootfs performs pivot_root and mounts /proc, /sys, /dev.
func setupRootfs(rootfsPath string) error {
	// Bind mount rootfs to itself
	if err := syscall.Mount(rootfsPath, rootfsPath, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("bind mount rootfs failed: %w", err)
	}

	if err := syscall.Chdir(rootfsPath); err != nil {
		return fmt.Errorf("chdir to rootfs failed: %w", err)
	}

	// pivot_root
	pivotDir := ".pivot_root"
	if err := unix.PivotRoot(".", pivotDir); err != nil {
		return fmt.Errorf("pivot_root failed: %w", err)
	}

	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir to / failed: %w", err)
	}

	// Unmount old root
	if err := unix.Unmount("/"+pivotDir, unix.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount old root failed: %w", err)
	}
	os.Remove("/" + pivotDir)

	// Mount /proc
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("mount /proc failed: %w", err)
	}

	// Mount /sys read-only
	if err := syscall.Mount("sysfs", "/sys", "sysfs", syscall.MS_RDONLY, ""); err != nil {
		return fmt.Errorf("mount /sys failed: %w", err)
	}

	// Setup /dev
	if err := fs.MountDevTmpfs(); err != nil {
		return fmt.Errorf("mount /dev failed: %w", err)
	}
	if err := fs.CreateDeviceNodes(); err != nil {
		return fmt.Errorf("create device nodes failed: %w", err)
	}

	// Set controlling terminal if TTY mode
	if os.Getenv("MINICONTAINER_TTY") == "1" {
		unix.IoctlSetInt(int(os.Stdin.Fd()), unix.TIOCSCTTY, 0)
	}

	return nil
}

// buildContainerEnv builds the environment for the container process.
func buildContainerEnv() []string {
	env := []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"TERM=xterm",
	}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "MINICONTAINER_ENV_") {
			env = append(env, strings.TrimPrefix(e, "MINICONTAINER_ENV_"))
		}
	}
	return env
}
