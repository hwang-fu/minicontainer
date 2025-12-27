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

	case "pull":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: minicontainer pull <image>")
			os.Exit(1)
		}
		cmd.RunPull(os.Args[2])

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
	fmt.Println("Commands:")
	fmt.Println("  run      Run a container")
	fmt.Println("  stop     Stop a running container")
	fmt.Println("  rm       Remove a stopped container")
	fmt.Println("  ps       List containers")
	fmt.Println("  prune    Remove stale overlay directories")
	fmt.Println("  images   List local images")
	fmt.Println("  rmi      Remove an image")
	fmt.Println("  import   Import a tarball as an image")
	fmt.Println("  pull     Pull an image from registry")
	fmt.Println("  version  Show version")
}
