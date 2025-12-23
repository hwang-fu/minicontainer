package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/hwang-fu/minicontainer/cmd"
	"github.com/hwang-fu/minicontainer/container"

	"golang.org/x/sys/unix"
)

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
		cfg, cmdArgs := cmd.ParseRunFlags(os.Args[2:])
		if len(cmdArgs) < 1 {
			fmt.Fprintln(os.Stderr, "usage: minicontainer run [flags] <command> [args...]")
			os.Exit(1)
		}

		if cfg.Interactive && cfg.AllocateTTY {
			// Interactive TTY mode: create PTY and relay I/O
			container.RunWithTTY(cfg, cmdArgs)
		} else {
			// Non-interactive mode: direct stdin/stdout passthrough
			container.RunWithoutTTY(cfg, cmdArgs)
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

			// Set controlling terminal (fixes "job control turned off" warning)
			if os.Getenv("MINICONTAINER_TTY") == "1" {
				unix.IoctlSetInt(int(os.Stdin.Fd()), unix.TIOCSCTTY, 0)
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
