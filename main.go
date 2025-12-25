package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/hwang-fu/minicontainer/cmd"
	"github.com/hwang-fu/minicontainer/container"
	"github.com/hwang-fu/minicontainer/fs"
	"github.com/hwang-fu/minicontainer/state"

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

		if cfg.RootfsPath == "" {
			fmt.Fprintln(os.Stderr, "error: --rootfs is required")
			fmt.Fprintln(os.Stderr, "usage: minicontainer run --rootfs <path> [flags] <command> [args...]")
			os.Exit(1)
		}

		if cfg.Detached {
			container.RunDetached(cfg, cmdArgs)
		} else if cfg.AllocateTTY {
			container.RunWithTTY(cfg, cmdArgs)
		} else {
			container.RunWithoutTTY(cfg, cmdArgs)
		}

	case "stop":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: minicontainer stop <container>")
			os.Exit(1)
		}

		cs, err := state.FindContainer(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		if cs.Status != state.StatusRunning {
			fmt.Fprintf(os.Stderr, "container %s is not running\n", cs.Name)
			os.Exit(1)
		}

		// Send SIGTERM first
		syscall.Kill(cs.PID, syscall.SIGTERM)

		// Wait briefly, then SIGKILL if still running
		time.Sleep(100 * time.Millisecond)
		syscall.Kill(cs.PID, syscall.SIGKILL)

		fmt.Println(container.ShortID(cs.ID))

	case "rm":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: minicontainer rm <container>")
			os.Exit(1)
		}

		cs, err := state.FindContainer(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		if cs.Status == state.StatusRunning {
			fmt.Fprintf(os.Stderr, "cannot remove running container %s, stop it first\n", cs.Name)
			os.Exit(1)
		}

		// Remove container directory (state.json and any other files)
		if err := os.RemoveAll(state.ContainerDir(cs.ID)); err != nil {
			fmt.Fprintf(os.Stderr, "failed to remove container: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(container.ShortID(cs.ID))

	case "ps":
		showAll := len(os.Args) > 2 && (os.Args[2] == "-a" || os.Args[2] == "--all")
		containers, err := state.ListContainers()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%-12s  %-20s  %-10s  %s\n", "CONTAINER ID", "COMMAND", "STATUS", "NAME")
		for _, c := range containers {
			if !showAll && c.Status != state.StatusRunning {
				continue
			}
			cmd := strings.Join(c.Command, " ")
			if len(cmd) > 20 {
				cmd = cmd[:17] + "..."
			}
			fmt.Printf("%-12s  %-20s  %-10s  %s\n",
				container.ShortID(c.ID), cmd, c.Status, c.Name)
		}

	case "prune":
		fmt.Println("Cleaning up stale overlay directories...")
		removed := fs.CleanupStaleOverlays()
		if len(removed) == 0 {
			fmt.Println("Nothing to clean.")
		} else {
			for _, dir := range removed {
				fmt.Printf("  Removed: %s\n", dir)
			}
			fmt.Printf("Removed %d directories.\n", len(removed))
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

		// Make all mounts private to this namespace
		// This prevents mount events from propagating to/from the host
		// Required for pivot_root to work correctly with user namespaces
		if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
			fmt.Fprintf(os.Stderr, "make mounts private failed: %v\n", err)
			os.Exit(1)
		}

		// pivot_root into container rootfs if specified
		// Unlike chroot, pivot_root actually swaps the root mount, preventing escape via file descriptors
		rootfsPath := os.Getenv("MINICONTAINER_ROOTFS")
		if rootfsPath != "" {
			// 1. Bind mount rootfs to itself (makes it a mount point - required for pivot_root)
			if err := syscall.Mount(rootfsPath, rootfsPath, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
				fmt.Fprintf(os.Stderr, "bind mount rootfs failed: %v\n", err)
				os.Exit(1)
			}

			// 2. Change to rootfs directory before pivot_root
			if err := syscall.Chdir(rootfsPath); err != nil {
				fmt.Fprintf(os.Stderr, "chdir to rootfs failed: %v\n", err)
				os.Exit(1)
			}

			// 3. Directory for old root (created by parent before namespace entry)
			pivotDir := ".pivot_root"

			// 4. pivot_root swaps the root mount
			// "." becomes new root, old root moves to ".pivot_root"
			if err := unix.PivotRoot(".", pivotDir); err != nil {
				fmt.Fprintf(os.Stderr, "pivot_root failed: %v\n", err)
				os.Exit(1)
			}

			// 5. Change to new root
			if err := syscall.Chdir("/"); err != nil {
				fmt.Fprintf(os.Stderr, "chdir to / failed: %v\n", err)
				os.Exit(1)
			}

			// 6. Unmount old root (MNT_DETACH allows unmount even if busy)
			if err := unix.Unmount("/"+pivotDir, unix.MNT_DETACH); err != nil {
				fmt.Fprintf(os.Stderr, "unmount old root failed: %v\n", err)
				os.Exit(1)
			}

			// 7. Remove the now-empty pivot directory (best effort - may fail in user namespace)
			os.Remove("/" + pivotDir)

			// 8. Mount fresh /proc for this PID namespace (must happen after pivot_root)
			// This makes 'ps' and /proc/* show only container processes
			if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
				fmt.Fprintf(os.Stderr, "failed to mount /proc: %v\n", err)
				os.Exit(1)
			}

			// 9. Mount /sys as read-only (exposes kernel info, read-only for security)
			if err := syscall.Mount("sysfs", "/sys", "sysfs", syscall.MS_RDONLY, ""); err != nil {
				fmt.Fprintf(os.Stderr, "failed to mount /sys: %v\n", err)
				os.Exit(1)
			}

			// 10. Setup /dev with essential device nodes
			if err := fs.MountDevTmpfs(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to mount /dev: %v\n", err)
				os.Exit(1)
			}
			if err := fs.CreateDeviceNodes(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to create device nodes: %v\n", err)
				os.Exit(1)
			}

			// Set controlling terminal (fixes "job control turned off" warning)
			if os.Getenv("MINICONTAINER_TTY") == "1" {
				unix.IoctlSetInt(int(os.Stdin.Fd()), unix.TIOCSCTTY, 0)
			}
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
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  run      Run a container")
	fmt.Println("  prune    Remove stale overlay directories")
	fmt.Println("  version  Show version")
}
