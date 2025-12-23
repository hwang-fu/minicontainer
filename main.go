package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// ContainerConfig holds the configuration options for a container.
// These are parsed from CLI flags in the run command and passed
// to the init process via environment variables.
type ContainerConfig struct {
	RootfsPath  string   // Path to container's root filesystem
	Hostname    string   // Custom hostname for the container
	Name        string   // Container name (for identification in ps, stop, etc.)
	Env         []string // User-specified environment variables (KEY=VALUE format)
	AutoRemove  bool     // If true, remove container filesystem on exit
	Interactive bool     // -i: Keep stdin open for interactive input
	AllocateTTY bool     // -t: Allocate pseudo-terminal for the container
}

// parseRunFlags parses command-line flags for the run command.
// It returns the parsed config and the remaining arguments (the command to run).
// Example: parseRunFlags(["--rootfs", "/tmp/alpine", "-e", "FOO=bar", "/bin/sh"])
// Returns: config{RootfsPath: "/tmp/alpine", Env: ["FOO=bar"]}, ["/bin/sh"]
func parseRunFlags(args []string) (ContainerConfig, []string) {
	cfg := ContainerConfig{}

	i := 0
	for i < len(args) {
		switch args[i] {
		case "--rootfs":
			// Container root filesystem path
			if i+1 < len(args) {
				cfg.RootfsPath = args[i+1]
				i += 2
			}

		case "--hostname":
			// Custom hostname for UTS namespace
			if i+1 < len(args) {
				cfg.Hostname = args[i+1]
				i += 2
			}

		case "--name":
			// Container name for later reference (ps, stop, rm)
			if i+1 < len(args) {
				cfg.Name = args[i+1]
				i += 2
			}

		case "-e", "--env":
			// Environment variable in KEY=VALUE format
			// Can be specified multiple times
			if i+1 < len(args) {
				cfg.Env = append(cfg.Env, args[i+1])
				i += 2
			}

		case "--rm":
			// Mark container for auto-removal on exit
			cfg.AutoRemove = true
			i++

		case "-i":
			// Interactive mode: keep stdin attached
			cfg.Interactive = true
			i++

		case "-t":
			// TTY mode: allocate pseudo-terminal
			cfg.AllocateTTY = true
			i++

		case "-it", "-ti":
			// Combined interactive + TTY (common shorthand)
			cfg.Interactive = true
			cfg.AllocateTTY = true
			i++

		default:
			// First non-flag argument is the command to run
			// Everything after is passed to that command
			return cfg, args[i:]
		}
	}
	return cfg, []string{}
}

// openPTY creates a new pseudo-terminal pair.
// Returns the master and slave file descriptors.
// The master is used by the parent (terminal side).
// The slave is used by the child (container side).
func openPTY() (*os.File, *os.File, error) {
	// Open the PTY master (multiplexor)
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open /dev/ptmx: %w", err)
	}

	// Get the slave PTY name and unlock it
	slaveName, err := ptsname(master)
	if err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("ptsname failed: %w", err)
	}

	if err := unlockpt(master); err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("unlockpt failed: %w", err)
	}

	// Open the slave PTY
	slave, err := os.OpenFile(slaveName, os.O_RDWR, 0)
	if err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("failed to open slave pty: %w", err)
	}

	return master, slave, nil
}

// ptsname returns the name of the slave pseudo-terminal device
// corresponding to the given master.
func ptsname(master *os.File) (string, error) {
	var n uint32
	// TIOCGPTN ioctl gets the slave pty number
	if err := unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCGPTN, uintptr(unsafe.Pointer(&n))); err != nil {
		return "", err
	}
	return fmt.Sprintf("/dev/pts/%d", n), nil
}

// unlockpt unlocks the slave pseudo-terminal device.
// Must be called before the slave can be opened.
func unlockpt(master *os.File) error {
	var unlock int
	// TIOCSPTLCK ioctl unlocks the slave pty (0 = unlock)
	return unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock)))
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "version":
		fmt.Println("minicontainer version 0.1.0")

	case "run":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: minicontainer run [flags] <command> [args...]")
			os.Exit(1)
		}

		// Parse CLI flags and extract the command to run
		cfg, cmdArgs := parseRunFlags(os.Args[2:])

		// Re-exec ourselves as "init" inside new namespaces
		// The init process will set up the container environment and exec the user command
		cmd := exec.Command("/proc/self/exe", append([]string{"init"}, cmdArgs...)...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Configure Linux namespaces for container isolation
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWUTS | // New hostname namespace
				syscall.CLONE_NEWPID | // New PID namespace (process is PID 1)
				syscall.CLONE_NEWIPC | // New IPC namespace (isolated shared memory, semaphores)
				syscall.CLONE_NEWUSER | // New user namespace (UID/GID mapping)
				syscall.CLONE_NEWNS, // New mount namespace (isolated mounts)
			// Map container root (UID 0) to current user on host
			// This allows unprivileged container operation
			UidMappings: []syscall.SysProcIDMap{
				{ContainerID: 0, HostID: os.Getuid(), Size: 1},
			},
			GidMappings: []syscall.SysProcIDMap{
				{ContainerID: 0, HostID: os.Getgid(), Size: 1},
			},
		}

		// Pass configuration to init process via environment variables
		// We use MINICONTAINER_ prefix to avoid conflicts with user env vars
		cmd.Env = os.Environ()
		if cfg.RootfsPath != "" {
			cmd.Env = append(cmd.Env, "MINICONTAINER_ROOTFS="+cfg.RootfsPath)
		}
		if cfg.Hostname != "" {
			cmd.Env = append(cmd.Env, "MINICONTAINER_HOSTNAME="+cfg.Hostname)
		}

		// Pass user-specified env vars with a prefix so init can extract them
		for _, e := range cfg.Env {
			cmd.Env = append(cmd.Env, "MINICONTAINER_ENV_"+e)
		}

		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	case "init":
		// init is the hidden command that runs INSIDE the container namespaces.
		// It sets up the container environment and then execs the user command.
		// This is the "re-exec" pattern used by Docker/runc.
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "init requires a command")
			os.Exit(1)
		}

		// Set container hostname (UTS namespace allows this without affecting host)
		// Use custom hostname if provided, otherwise default to "minicontainer"
		hostname := os.Getenv("MINICONTAINER_HOSTNAME")
		if hostname == "" {
			hostname = "minicontainer"
		}
		if err := syscall.Sethostname([]byte(hostname)); err != nil {
			fmt.Fprintf(os.Stderr, "failed to set hostname: %v\n", err)
			os.Exit(1)
		}

		// Chroot into container rootfs if specified
		// This changes the root directory so container can't see host filesystem
		rootfsPath := os.Getenv("MINICONTAINER_ROOTFS")
		if rootfsPath != "" {
			if err := syscall.Chroot(rootfsPath); err != nil {
				fmt.Fprintf(os.Stderr, "chroot failed: %v\n", err)
				os.Exit(1)
			}
			// Must chdir after chroot to actually enter the new root
			if err := syscall.Chdir("/"); err != nil {
				fmt.Fprintf(os.Stderr, "chdir failed: %v\n", err)
				os.Exit(1)
			}
		}

		// Mount fresh /proc for this PID namespace
		// This makes 'ps' and /proc/* show only container processes
		if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
			fmt.Fprintf(os.Stderr, "failed to mount /proc: %v\n", err)
			os.Exit(1)
		}

		// Find absolute path of the command to execute
		path, err := exec.LookPath(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "command not found: %s\n", os.Args[2])
			os.Exit(1)
		}

		// Build container environment
		// Start with minimal base environment
		env := []string{
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			"TERM=xterm",
		}
		// Add user-specified environment variables
		// These were passed from run command with MINICONTAINER_ENV_ prefix
		for _, e := range os.Environ() {
			if strings.HasPrefix(e, "MINICONTAINER_ENV_") {
				// Strip the prefix and add the actual env var
				env = append(env, strings.TrimPrefix(e, "MINICONTAINER_ENV_"))
			}
		}

		// Replace this process with the user command
		// Using syscall.Exec (not exec.Command) so the command becomes PID 1
		if err := syscall.Exec(path, os.Args[2:], env); err != nil {
			fmt.Fprintf(os.Stderr, "exec failed: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: ./minicontainer <command> [options]")
}
