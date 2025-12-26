package container

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/hwang-fu/minicontainer/cgroup"
	"github.com/hwang-fu/minicontainer/cmd"
	"github.com/hwang-fu/minicontainer/fs"
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
	CgroupPath     string                // Path to container's cgroup
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
		containerName = state.ShortID(containerID)
	}

	// Create initial state with status=created and save to disk
	containerState := state.NewContainerState(containerID, containerName, cfg.RootfsPath, cmdArgs)
	if err = state.SaveState(containerState); err != nil {
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

	// Cgroup creation
	cgroupPath, err := cgroup.CreateContainerCgroup(containerID)
	if err != nil {
		return nil, fmt.Errorf("create cgroup: %w", err)
	}
	if err := cgroup.ApplyResourceLimits(cgroupPath, cfg.MemoryLimit, cfg.CPULimit, cfg.PidsLimit); err != nil {
		return nil, err
	}
	cr.CgroupPath = cgroupPath

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

// NewNamespaceSysProcAttr creates SysProcAttr with Linux namespace flags.
// This configures the child process to run in isolated namespaces.
//
// Namespaces enabled:
//   - CLONE_NEWUTS: Own hostname
//   - CLONE_NEWPID: Own PID namespace (process sees itself as PID 1)
//   - CLONE_NEWIPC: Own IPC namespace
//   - CLONE_NEWNS: Own mount namespace
//   - CLONE_NEWUSER: User namespace (only when running as non-root)
//
// The setsid parameter controls whether to create a new session (needed for TTY).
func NewNamespaceSysProcAttr(setsid bool) *syscall.SysProcAttr {
	cloneFlags := syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWIPC | syscall.CLONE_NEWNS

	attr := &syscall.SysProcAttr{
		Cloneflags: uintptr(cloneFlags),
		Setsid:     setsid,
	}

	// User namespace is only used when running as non-root.
	// Maps container root (UID 0) to current host user.
	if os.Getuid() != 0 {
		attr.Cloneflags |= syscall.CLONE_NEWUSER
		attr.UidMappings = []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getuid(), Size: 1}}
		attr.GidMappings = []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getgid(), Size: 1}}
	}

	return attr
}

// BuildCommand creates the exec.Cmd for the container.
// Uses re-exec pattern: runs /proc/self/exe with "init" subcommand.
// The init handler (in main.go) then sets up the container environment.
//
// The tty parameter affects:
//   - Setsid flag (needed for TTY to work properly)
//   - MINICONTAINER_TTY environment variable
func (cr *ContainerRuntime) BuildCommand(tty bool) *exec.Cmd {
	// Re-exec ourselves with "init" subcommand - this is the industry standard pattern
	// used by Docker/runc. The child process will set up namespaces and exec the user command.
	execCmd := exec.Command("/proc/self/exe", append([]string{"init"}, cr.CmdArgs...)...)
	execCmd.SysProcAttr = NewNamespaceSysProcAttr(tty)

	// Pass config to init via environment variables, using overlayfs merged path
	cfgWithOverlay := cr.Config
	cfgWithOverlay.RootfsPath = cr.ActualRootfs
	execCmd.Env = BuildEnv(cfgWithOverlay)
	if tty {
		execCmd.Env = append(execCmd.Env, "MINICONTAINER_TTY=1")
	}

	cr.Cmd = execCmd
	return execCmd
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

// AddToCgroup adds the container process to its cgroup.
// Must be called after cmd.Start() when we have the PID.
func (cr *ContainerRuntime) AddToCgroup() error {
	if cr.CgroupPath == "" {
		return nil // No cgroup configured
	}
	return cgroup.AddProcessToCgroup(cr.CgroupPath, cr.Cmd.Process.Pid)
}

// Cleanup cleans up overlay filesystem.
func (cr *ContainerRuntime) Cleanup() {
	if cr.OverlayCleanup != nil {
		cr.OverlayCleanup()
	}
}
