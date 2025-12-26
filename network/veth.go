package network

import "fmt"

// CreateVethPair creates a veth pair and attaches host end to bridge.
// Returns (hostVeth, containerVeth) names.
// containerVeth will be moved into container's netns and renamed to eth0.
func CreateVethPair(containerID string) (hostVeth string, containerVeth string, err error) {
	// Use short ID for veth names (max 15 chars for interface names)
	shortID := containerID[:8]
	hostVeth = "veth-" + shortID
	containerVeth = "veth-c-" + shortID

	// Create veth pair
	if err := run("ip", "link", "add", hostVeth, "type", "veth", "peer", "name", containerVeth); err != nil {
		return "", "", fmt.Errorf("create veth pair: %w", err)
	}

	// Attach host end to bridge
	if err := run("ip", "link", "set", hostVeth, "master", BridgeName); err != nil {
		return "", "", fmt.Errorf("attach to bridge: %w", err)
	}

	// Bring host end up
	if err := run("ip", "link", "set", hostVeth, "up"); err != nil {
		return "", "", fmt.Errorf("bring veth up: %w", err)
	}

	return hostVeth, containerVeth, nil
}

// MoveVethToNetns moves the container-side veth into a network namespace.
// pid is the container's init process PID.
func MoveVethToNetns(containerVeth string, pid int) error {
	if err := run("ip", "link", "set", containerVeth, "netns", fmt.Sprintf("%d", pid)); err != nil {
		return fmt.Errorf("move veth to netns: %w", err)
	}
	return nil
}
