package cmd

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

	fmt.Println(container.ShortID(cs.ID))
}
