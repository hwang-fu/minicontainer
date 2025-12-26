package network

import (
	"fmt"
	"os"
	"os/exec"
)

const (
	BridgeName = "minicontainer0"
	BridgeCIDR = "172.17.0.1/16"
)

// EnsureBridge creates the minicontainer0 bridge if it doesn't exist.
// Assigns 172.17.0.1/16 as the bridge IP (gateway for containers).
func EnsureBridge() error {
	// Check if bridge already exists
	if _, err := net.InterfaceByName(BridgeName); err == nil {
		return nil // Already exists
	}

	// Create bridge using ip command
	if err := run("ip", "link", "add", BridgeName, "type", "bridge"); err != nil {
		return fmt.Errorf("create bridge: %w", err)
	}

	// Assign IP address
	if err := run("ip", "addr", "add", BridgeCIDR, "dev", BridgeName); err != nil {
		return fmt.Errorf("assign bridge IP: %w", err)
	}

	// Bring bridge up
	if err := run("ip", "link", "set", BridgeName, "up"); err != nil {
		return fmt.Errorf("bring bridge up: %w", err)
	}

	return nil
}

// run executes a command and returns any error.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
