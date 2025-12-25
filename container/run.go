package container

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/hwang-fu/minicontainer/cmd"
	"github.com/hwang-fu/minicontainer/fs"
	"github.com/hwang-fu/minicontainer/runtime"
	"github.com/hwang-fu/minicontainer/state"
)

// ContainerRuntime holds common runtime state
type ContainerRuntime struct {
	ID             string                // Full 64-char container ID
	Name           string                // Display name (user-provided or short ID)
	Config         cmd.ContainerConfig   // Original config from CLI flags
	CmdArgs        []string              // Command and arguments to run in container
	State          *state.ContainerState // Persistent state (saved to disk)
	ActualRootfs   string                // Path to merged overlayfs (or original rootfs if no overlay)
	OverlayCleanup func() error          // Cleanup function for overlayfs (nil if no overlay)
	Cmd            *exec.Cmd             // The exec.Cmd for the container process
}

// NewContainerRuntime initializes a container: generates ID, creates state, sets up overlay.
// This is the common setup shared by all run modes.
// Returns error if any step fails; caller should handle cleanup.
func NewContainerRuntime(cfg cmd.ContainerConfig, cmdArgs []string) (*ContainerRuntime, error) {
	// Prepare rootfs directories before namespace entry (avoids permission issues)
	if err := prepareRootfs(cfg.RootfsPath); err != nil {
		return nil, fmt.Errorf("prepare rootfs: %w", err)
	}

	// Generate unique 64-char hex ID using SHA256 of random bytes
	containerID, err := GenerateContainerID()
	if err != nil {
		return nil, fmt.Errorf("generate container ID: %w", err)
	}

	// Use provided name or default to short ID (first 12 chars)
	containerName := cfg.Name
	if containerName == "" {
		containerName = ShortID(containerID)
	}

	// Create initial state with status=created and save to disk
	containerState := state.NewContainerState(containerID, containerName, cfg.RootfsPath, cmdArgs)
	if err := state.SaveState(containerState); err != nil {
		return nil, fmt.Errorf("save state: %w", err)
	}

	cr := &ContainerRuntime{
		ID:           containerID,
		Name:         containerName,
		Config:       cfg,
		CmdArgs:      cmdArgs,
		State:        containerState,
		ActualRootfs: cfg.RootfsPath,
	}

	// Setup overlayfs: lower=rootfs (read-only), upper=writable layer, merged=container view
	if cfg.RootfsPath != "" {
		overlay, cleanup, err := fs.SetupOverlayfs(cfg.RootfsPath)
		if err != nil {
			return nil, fmt.Errorf("setup overlay: %w", err)
		}
		cr.OverlayCleanup = cleanup
		cr.ActualRootfs = overlay.MergedDir
	}

	// Bind mount volumes into container rootfs (must happen before pivot_root)
	if len(cfg.Volumes) > 0 && cr.ActualRootfs != "" {
		if err := fs.MountVolumes(cr.ActualRootfs, cfg.Volumes); err != nil {
			cr.Cleanup()
			return nil, fmt.Errorf("mount volumes: %w", err)
		}
	}

	return cr, nil
}

// MarkRunning updates state to running with PID.
func (cr *ContainerRuntime) MarkRunning() {
	cr.State.PID = cr.Cmd.Process.Pid
	cr.State.Status = state.StatusRunning
	state.SaveState(cr.State)
}

// MarkStopped updates state to stopped with exit code.
func (cr *ContainerRuntime) MarkStopped() {
	cr.State.Status = state.StatusStopped
	cr.State.ExitCode = getExitCode(cr.Cmd.ProcessState)
	state.SaveState(cr.State)
}

// ForwardSignals forwards SIGINT/SIGTERM to container process.
func (cr *ContainerRuntime) ForwardSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range sigChan {
			syscall.Kill(cr.Cmd.Process.Pid, sig.(syscall.Signal))
		}
	}()
}

// Cleanup cleans up overlay filesystem.
func (cr *ContainerRuntime) Cleanup() {
	if cr.OverlayCleanup != nil {
		cr.OverlayCleanup()
	}
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
	// Pass volume specifications to init
	for i, v := range cfg.Volumes {
		env = append(env, fmt.Sprintf("MINICONTAINER_VOLUME_%d=%s", i, v))
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

	// Generate container ID and create initial state
	containerID, err := GenerateContainerID()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate container ID: %v\n", err)
		os.Exit(1)
	}
	containerName := cfg.Name
	if containerName == "" {
		containerName = ShortID(containerID)
	}
	containerState := state.NewContainerState(containerID, containerName, cfg.RootfsPath, cmdArgs)
	if err = state.SaveState(containerState); err != nil {
		fmt.Fprintf(os.Stderr, "failed to save state: %v\n", err)
		os.Exit(1)
	}
	// fmt.Printf("%s\n", containerID)  // Only print in detached mode

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

	// Setup overlayfs if rootfs is specified
	var overlayCleanup func() error
	actualRootfs := cfg.RootfsPath
	if cfg.RootfsPath != "" {
		overlay, cleanup, err := fs.SetupOverlayfs(cfg.RootfsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to setup overlay: %v\n", err)
			os.Exit(1)
		}
		overlayCleanup = cleanup
		actualRootfs = overlay.MergedDir
	}

	// Mount volumes into the container rootfs
	if len(cfg.Volumes) > 0 && actualRootfs != "" {
		if err := fs.MountVolumes(actualRootfs, cfg.Volumes); err != nil {
			fmt.Fprintf(os.Stderr, "failed to mount volumes: %v\n", err)
			if overlayCleanup != nil {
				overlayCleanup()
			}
			os.Exit(1)
		}
	}

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

	cfgWithOverlay := cfg
	cfgWithOverlay.RootfsPath = actualRootfs
	cmd.Env = append(BuildEnv(cfgWithOverlay), "MINICONTAINER_TTY=1")

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	containerState.PID = cmd.Process.Pid
	containerState.Status = state.StatusRunning
	state.SaveState(containerState)

	// Forward signals to container
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range sigChan {
			syscall.Kill(cmd.Process.Pid, sig.(syscall.Signal))
		}
	}()

	slave.Close() // Close slave in parent after child starts

	// Relay I/O: PTY master -> stdout (always)
	go io.Copy(os.Stdout, master)
	// Only relay stdin -> PTY master if interactive mode
	if cfg.Interactive {
		go io.Copy(master, os.Stdin)
	}

	cmd.Wait()

	containerState.Status = state.StatusStopped
	containerState.ExitCode = getExitCode(cmd.ProcessState)
	state.SaveState(containerState)

	if overlayCleanup != nil {
		overlayCleanup()
	}

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

	// Generate container ID and create initial state
	containerID, err := GenerateContainerID()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate container ID: %v\n", err)
		os.Exit(1)
	}
	containerName := cfg.Name
	if containerName == "" {
		containerName = ShortID(containerID)
	}
	containerState := state.NewContainerState(containerID, containerName, cfg.RootfsPath, cmdArgs)
	if err = state.SaveState(containerState); err != nil {
		fmt.Fprintf(os.Stderr, "failed to save state: %v\n", err)
		os.Exit(1)
	}
	// fmt.Printf("%s\n", containerID)  // Only print in detached mode

	// Setup overlayfs if rootfs is specified
	var overlayCleanup func() error
	actualRootfs := cfg.RootfsPath
	if cfg.RootfsPath != "" {
		overlay, cleanup, err := fs.SetupOverlayfs(cfg.RootfsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to setup overlay: %v\n", err)
			os.Exit(1)
		}
		overlayCleanup = cleanup
		actualRootfs = overlay.MergedDir
	}

	// Mount volumes into the container rootfs
	if len(cfg.Volumes) > 0 && actualRootfs != "" {
		if err := fs.MountVolumes(actualRootfs, cfg.Volumes); err != nil {
			fmt.Fprintf(os.Stderr, "failed to mount volumes: %v\n", err)
			if overlayCleanup != nil {
				overlayCleanup()
			}
			os.Exit(1)
		}
	}

	cmd := exec.Command("/proc/self/exe", append([]string{"init"}, cmdArgs...)...)
	// Only connect stdin if interactive mode (-i flag)
	if cfg.Interactive {
		cmd.Stdin = os.Stdin
	}
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

	cfgWithOverlay := cfg
	cfgWithOverlay.RootfsPath = actualRootfs
	cmd.Env = BuildEnv(cfgWithOverlay)

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		if overlayCleanup != nil {
			overlayCleanup()
		}
		os.Exit(1)
	}

	containerState.PID = cmd.Process.Pid
	containerState.Status = state.StatusRunning
	state.SaveState(containerState)

	// Forward signals to container
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range sigChan {
			syscall.Kill(cmd.Process.Pid, sig.(syscall.Signal))
		}
	}()

	cmd.Wait()

	containerState.Status = state.StatusStopped
	containerState.ExitCode = getExitCode(cmd.ProcessState)
	state.SaveState(containerState)

	if overlayCleanup != nil {
		overlayCleanup()
	}
}

// RunDetached runs container in background, returns immediately.
func RunDetached(cfg cmd.ContainerConfig, cmdArgs []string) {
	if err := prepareRootfs(cfg.RootfsPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to prepare rootfs: %v\n", err)
		os.Exit(1)
	}

	containerID, err := GenerateContainerID()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate container ID: %v\n", err)
		os.Exit(1)
	}
	containerName := cfg.Name
	if containerName == "" {
		containerName = ShortID(containerID)
	}
	containerState := state.NewContainerState(containerID, containerName, cfg.RootfsPath, cmdArgs)
	if err = state.SaveState(containerState); err != nil {
		fmt.Fprintf(os.Stderr, "failed to save state: %v\n", err)
		os.Exit(1)
	}

	// Setup overlayfs
	var overlayCleanup func() error
	actualRootfs := cfg.RootfsPath
	if cfg.RootfsPath != "" {
		overlay, cleanup, err := fs.SetupOverlayfs(cfg.RootfsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to setup overlay: %v\n", err)
			os.Exit(1)
		}
		overlayCleanup = cleanup
		_ = overlayCleanup // Will be cleaned up when container exits
		actualRootfs = overlay.MergedDir
	}

	execCmd := exec.Command("/proc/self/exe", append([]string{"init"}, cmdArgs...)...)
	execCmd.Stdin = nil
	execCmd.Stdout = nil
	execCmd.Stderr = nil

	cloneFlags := syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWIPC | syscall.CLONE_NEWNS
	execCmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: uintptr(cloneFlags),
		Setsid:     true,
	}

	cfgWithOverlay := cfg
	cfgWithOverlay.RootfsPath = actualRootfs
	execCmd.Env = BuildEnv(cfgWithOverlay)

	if err := execCmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	containerState.PID = execCmd.Process.Pid
	containerState.Status = state.StatusRunning
	state.SaveState(containerState)

	// Detach - don't wait
	fmt.Println(containerID)
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
