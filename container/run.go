package container

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hwang-fu/minicontainer/cmd"
	"github.com/hwang-fu/minicontainer/runtime"
)

// RunWithTTY runs the container with pseudo-terminal for interactive mode.
// Creates PTY, sets raw mode, and relays I/O between terminal and container.
func RunWithTTY(cfg cmd.ContainerConfig, cmdArgs []string) {
	cr, err := NewContainerRuntime(cfg, cmdArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize container: %v\n", err)
		os.Exit(1)
	}

	// Create PTY pair: master (host side), slave (container side)
	master, slave, err := runtime.OpenPTY()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create pty: %v\n", err)
		os.Exit(1)
	}
	defer master.Close()
	defer slave.Close()

	// Set terminal to raw mode, get restore function
	restoreFunc, err := runtime.SetRawMode(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set raw mode: %v\n", err)
		os.Exit(1)
	}
	defer restoreFunc()

	// Build and configure command
	execCmd := cr.BuildCommand(true) // tty=true
	execCmd.Stdin = slave
	execCmd.Stdout = slave
	execCmd.Stderr = slave

	if err := execCmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cr.MarkRunning()
	cr.ForwardSignals()
	slave.Close() // Close slave in parent after child starts

	// Relay I/O between terminal and PTY
	go io.Copy(os.Stdout, master)
	if cfg.Interactive {
		go io.Copy(master, os.Stdin)
	}

	execCmd.Wait()
	cr.MarkStopped()
	cr.Cleanup()

	master.Close()
	restoreFunc()
}

// RunDetached runs container in background, returns immediately.
func RunDetached(cfg cmd.ContainerConfig, cmdArgs []string) {
	cr, err := NewContainerRuntime(cfg, cmdArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize container: %v\n", err)
		os.Exit(1)
	}

	// Build command for detached mode (no stdin/stdout)
	execCmd := cr.BuildCommand(false)
	execCmd.Stdin = nil
	execCmd.Stdout = nil
	execCmd.Stderr = nil

	if err := execCmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cr.MarkRunning()

	// Print ID and exit - don't wait for container
	fmt.Println(cr.ID)
}

// RunWithoutTTY runs container with direct stdin/stdout passthrough.
func RunWithoutTTY(cfg cmd.ContainerConfig, cmdArgs []string) {
	cr, err := NewContainerRuntime(cfg, cmdArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize container: %v\n", err)
		os.Exit(1)
	}

	// Build command without TTY
	execCmd := cr.BuildCommand(false)
	if cfg.Interactive {
		execCmd.Stdin = os.Stdin
	}
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		cr.Cleanup()
		os.Exit(1)
	}

	cr.MarkRunning()
	cr.ForwardSignals()

	execCmd.Wait()
	cr.MarkStopped()
	cr.Cleanup()
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
	for i, v := range cfg.Volumes {
		env = append(env, fmt.Sprintf("MINICONTAINER_VOLUME_%d=%s", i, v))
	}
	return env
}

// prepareRootfs creates necessary directories inside rootfs before namespace entry.
// This must be done in the parent process to avoid permission issues in user namespace.
func prepareRootfs(rootfsPath string) error {
	if rootfsPath == "" {
		return nil
	}
	// Create .pivot_root directory for pivot_root syscall
	if err := os.MkdirAll(filepath.Join(rootfsPath, ".pivot_root"), 0o700); err != nil {
		return err
	}
	// Create mount points for virtual filesystems
	for _, dir := range []string{"proc", "sys"} {
		if err := os.MkdirAll(filepath.Join(rootfsPath, dir), 0o755); err != nil {
			return err
		}
	}
	return nil
}

// getExitCode extracts the exit code from a process state.
func getExitCode(processState *os.ProcessState) int {
	if processState == nil {
		return -1
	}
	return processState.ExitCode()
}
