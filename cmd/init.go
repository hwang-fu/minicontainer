package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
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
