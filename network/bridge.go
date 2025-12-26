package network

import (
	"os"
	"os/exec"
)

// run executes a command and returns any error.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
