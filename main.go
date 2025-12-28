package main

import (
	"fmt"
	"os"

	"github.com/hwang-fu/minicontainer/cmd"
	"github.com/hwang-fu/minicontainer/container"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {

	case "help", "--help", "-h":
		printUsage()
		os.Exit(0)

	case "--version", "-v":
		fmt.Println("minicontainer version 0.1.0")
		os.Exit(0)

	case "version":
		fmt.Println("minicontainer version 0.1.0")

	case "run":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: minicontainer run [flags] <image|--rootfs path> [command] [args...]")
			os.Exit(1)
		}

		// Parse CLI flags and extract the command to run
		cfg, cmdArgs := cmd.ParseRunFlags(os.Args[2:])

		// Resolve rootfs from --rootfs flag or image reference
		resolvedCfg, cmdArgs, err := cmd.ResolveRootfs(&cfg, cmdArgs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		if len(cmdArgs) < 1 {
			fmt.Fprintln(os.Stderr, "error: no command specified")
			os.Exit(1)
		}

		if resolvedCfg.Detached {
			container.RunDetached(*resolvedCfg, cmdArgs)
		} else if resolvedCfg.AllocateTTY {
			container.RunWithTTY(*resolvedCfg, cmdArgs)
		} else {
			container.RunWithoutTTY(*resolvedCfg, cmdArgs)
		}

	case "exec":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: minicontainer exec [options] <container> <command> [args...]")
			os.Exit(1)
		}
		cmd.RunExec(os.Args[2:])

	case "stop":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: minicontainer stop <container>")
			os.Exit(1)
		}
		cmd.RunStop(os.Args[2])

	case "rm":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: minicontainer rm <container>")
			fmt.Fprintln(os.Stderr, "       minicontainer rm -a | --all")
			os.Exit(1)
		}

		if os.Args[2] == "--all" || os.Args[2] == "-a" {
			cmd.RunRmAll()
		} else {
			cmd.RunRm(os.Args[2])
		}

	case "ps":
		showAll := len(os.Args) > 2 && (os.Args[2] == "-a" || os.Args[2] == "--all")
		cmd.RunPs(showAll)

	case "prune":
		cmd.RunPrune()

	case "images":
		cmd.RunImages()

	case "rmi":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: minicontainer rmi <image>")
			os.Exit(1)
		}
		cmd.RunRmi(os.Args[2])

	case "import":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: minicontainer import <tarball> <name[:tag]>")
			os.Exit(1)
		}
		cmd.RunImport(os.Args[2], os.Args[3])

	case "inspect":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: minicontainer inspect <container>")
			os.Exit(1)
		}
		cmd.RunInspect(os.Args[2])

	case "pull":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: minicontainer pull <image>")
			os.Exit(1)
		}
		cmd.RunPull(os.Args[2])

	case "logs":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: minicontainer logs <container>")
			os.Exit(1)
		}
		cmd.RunLogs(os.Args[2])

	case "init":
		cmd.RunInit(os.Args[2:])

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: minicontainer <command> [options]")
	fmt.Println()
	fmt.Println("Container Commands:")
	fmt.Println("  run      Create and run a container")
	fmt.Println("  exec     Execute a command in a running container")
	fmt.Println("  stop     Stop a running container")
	fmt.Println("  rm       Remove a stopped container")
	fmt.Println("  ps       List containers")
	fmt.Println("  logs     Fetch the logs of a container")
	fmt.Println("  inspect  Display detailed container information")
	fmt.Println()
	fmt.Println("Image Commands:")
	fmt.Println("  images   List local images")
	fmt.Println("  pull     Pull an image from a registry")
	fmt.Println("  import   Import a tarball as an image")
	fmt.Println("  rmi      Remove an image")
	fmt.Println()
	fmt.Println("Other Commands:")
	fmt.Println("  prune    Remove stale overlay directories")
	fmt.Println("  version  Show version information")
}

func printCommandHelp(command string) {
	switch command {
	case "run":
		fmt.Println("Usage: minicontainer run [options] <image|--rootfs path> <command> [args...]")
		fmt.Println()
		fmt.Println("Create and run a container")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --rootfs PATH         Container root filesystem")
		fmt.Println("  --name NAME           Container name")
		fmt.Println("  --hostname NAME       Container hostname")
		fmt.Println("  -d, --detach          Run in background")
		fmt.Println("  -i, --interactive     Keep stdin open")
		fmt.Println("  -t, --tty             Allocate pseudo-TTY")
		fmt.Println("  -e, --env KEY=VAL     Set environment variable")
		fmt.Println("  -v, --volume SRC:DST  Bind mount volume")
		fmt.Println("  -p, --publish H:C     Publish port (host:container)")
		fmt.Println("  --memory SIZE         Memory limit (e.g., 256m, 1g)")
		fmt.Println("  --cpus N              CPU limit (e.g., 0.5, 2)")
		fmt.Println("  --pids-limit N        Max number of processes")
	case "exec":
		fmt.Println("Usage: minicontainer exec <container> <command> [args...]")
		fmt.Println()
		fmt.Println("Execute a command in a running container")
	case "stop":
		fmt.Println("Usage: minicontainer stop <container>")
		fmt.Println()
		fmt.Println("Stop a running container")
	case "rm":
		fmt.Println("Usage: minicontainer rm <container>")
		fmt.Println("       minicontainer rm -a|--all")
		fmt.Println()
		fmt.Println("Remove a stopped container")
	case "ps":
		fmt.Println("Usage: minicontainer ps [-a|--all]")
		fmt.Println()
		fmt.Println("List containers (default: running only)")
	case "logs":
		fmt.Println("Usage: minicontainer logs <container>")
		fmt.Println()
		fmt.Println("Fetch the logs of a container")
	case "inspect":
		fmt.Println("Usage: minicontainer inspect <container>")
		fmt.Println()
		fmt.Println("Display detailed container information as JSON")
	case "pull":
		fmt.Println("Usage: minicontainer pull <image>")
		fmt.Println()
		fmt.Println("Pull an image from a registry")
	case "images":
		fmt.Println("Usage: minicontainer images")
		fmt.Println()
		fmt.Println("List local images")
	case "rmi":
		fmt.Println("Usage: minicontainer rmi <image>")
		fmt.Println()
		fmt.Println("Remove an image")
	case "import":
		fmt.Println("Usage: minicontainer import <tarball> <name[:tag]>")
		fmt.Println()
		fmt.Println("Import a tarball as an image")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
	}
}
