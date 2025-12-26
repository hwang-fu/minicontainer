package network

import "fmt"

// SetupNAT configures iptables for container outbound connectivity.
func SetupNAT() error {
	// Enable IP forwarding
	if err := run("sysctl", "-w", "net.ipv4.ip_forward=1"); err != nil {
		return fmt.Errorf("enable ip forward: %w", err)
	}

	// Masquerade traffic from container subnet
	if err := run("iptables", "-t", "nat", "-A", "POSTROUTING",
		"-s", "172.18.0.0/16", "!", "-o", BridgeName, "-j", "MASQUERADE"); err != nil {
		return fmt.Errorf("add masquerade rule: %w", err)
	}

	// Allow forwarding to/from bridge
	run("iptables", "-A", "FORWARD", "-i", BridgeName, "-j", "ACCEPT")
	run("iptables", "-A", "FORWARD", "-o", BridgeName, "-j", "ACCEPT")

	return nil
}
