package container

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
)

// BuildEnv creates environment variables to pass to init process.
func BuildEnv(cfg ContainerConfig) []string {
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
func RunWithTTY(cfg ContainerConfig, cmdArgs []string) {
	master, slave, err := OpenPTY()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create pty: %v\n", err)
		os.Exit(1)
	}
	defer master.Close()
	defer slave.Close()

	// SetRawMode returns (restoreFunc, error) - restoreFunc resets terminal on exit
	restoreFunc, err := SetRawMode(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set raw mode: %v\n", err)
		os.Exit(1)
	}
	defer restoreFunc()

	cmd := exec.Command("/proc/self/exe", append([]string{"init"}, cmdArgs...)...)
	cmd.Stdin = slave
	cmd.Stdout = slave
	cmd.Stderr = slave
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:  syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWIPC | syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getuid(), Size: 1}},
		GidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getgid(), Size: 1}},
		Setsid:      true,
	}
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
func RunWithoutTTY(cfg ContainerConfig, cmdArgs []string) {
	cmd := exec.Command("/proc/self/exe", append([]string{"init"}, cmdArgs...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID |
			syscall.CLONE_NEWIPC | syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getuid(), Size: 1}},
		GidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getgid(), Size: 1}},
	}
	cmd.Env = BuildEnv(cfg)

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
