package network

import (
	"os"
	"os/exec"
)

const (
	BridgeName = "minicontainer0"
	BridgeCIDR = "172.17.0.1/16"
)

// run executes a command and returns any error.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
