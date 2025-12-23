package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
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
			fmt.Fprintln(os.Stderr, "usage: minicontainer run <command> [args...]")
			os.Exit(1)
		}

		cmd := exec.Command("/proc/self/exe", append([]string{"init"}, os.Args[2:]...)...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWUTS |
				syscall.CLONE_NEWPID |
				syscall.CLONE_NEWIPC |
				syscall.CLONE_NEWUSER |
				syscall.CLONE_NEWNS,
			UidMappings: []syscall.SysProcIDMap{
				{ContainerID: 0, HostID: os.Getuid(), Size: 1},
			},
			GidMappings: []syscall.SysProcIDMap{
				{ContainerID: 0, HostID: os.Getgid(), Size: 1},
			},
		}

		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	case "init":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "init requires a command")
			os.Exit(1)
		}

		if err := syscall.Sethostname([]byte("minicontainer")); err != nil {
			fmt.Fprintf(os.Stderr, "failed to set hostname: %v\n", err)
			os.Exit(1)
		}

		// Find the absolute path of the command
		path, err := exec.LookPath(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "command not found: %s\n", os.Args[2])
			os.Exit(1)
		}

		// Mount fresh /proc for this PID namespace
		if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
			fmt.Fprintf(os.Stderr, "failed to mount /proc: %v\n", err)
			os.Exit(1)
		}

		if err := syscall.Exec(path, os.Args[2:], os.Environ()); err != nil {
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
