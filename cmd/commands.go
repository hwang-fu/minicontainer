package cmd

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/hwang-fu/minicontainer/state"
)

// RunStop stops a running container.
func RunStop(idOrName string) {
	cs, err := state.FindContainer(idOrName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if cs.Status != state.StatusRunning {
		fmt.Fprintf(os.Stderr, "container %s is not running\n", cs.Name)
		os.Exit(1)
	}

	syscall.Kill(cs.PID, syscall.SIGTERM)
	time.Sleep(100 * time.Millisecond)
	syscall.Kill(cs.PID, syscall.SIGKILL)

	fmt.Println(state.ShortID(cs.ID))
}

// RunRm removes a stopped container.
func RunRm(idOrName string) {
	cs, err := state.FindContainer(idOrName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if cs.Status == state.StatusRunning {
		fmt.Fprintf(os.Stderr, "cannot remove running container %s, stop it first\n", cs.Name)
		os.Exit(1)
	}

	cgroup.RemoveContainerCgroup(cs.ID)
	if err := os.RemoveAll(state.ContainerDir(cs.ID)); err != nil {
		fmt.Fprintf(os.Stderr, "failed to remove container: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(state.ShortID(cs.ID))
}
