package network

import "fmt"

// SetupContainerNetwork configures networking inside the container's netns.
// Must be called from host after moving veth into container's netns.
// Uses nsenter to run commands in the container's network namespace.
func SetupContainerNetwork(pid int, vethName string, ip string) error {
	ns := fmt.Sprintf("%d", pid)

	// Rename veth to eth0
	if err := run("nsenter", "-t", ns, "-n", "--", "ip", "link", "set", vethName, "name", "eth0"); err != nil {
		return fmt.Errorf("rename veth to eth0: %w", err)
	}

	// Assign IP address
	if err := run("nsenter", "-t", ns, "-n", "--", "ip", "addr", "add", ip+"/16", "dev", "eth0"); err != nil {
		return fmt.Errorf("assign IP: %w", err)
	}

	// Bring eth0 up
	if err := run("nsenter", "-t", ns, "-n", "--", "ip", "link", "set", "eth0", "up"); err != nil {
		return fmt.Errorf("bring eth0 up: %w", err)
	}

	// Bring loopback up
	if err := run("nsenter", "-t", ns, "-n", "--", "ip", "link", "set", "lo", "up"); err != nil {
		return fmt.Errorf("bring lo up: %w", err)
	}

	// Add default route via bridge
	if err := run("nsenter", "-t", ns, "-n", "--", "ip", "route", "add", "default", "via", Gateway()); err != nil {
		return fmt.Errorf("add default route: %w", err)
	}

	return nil
}
