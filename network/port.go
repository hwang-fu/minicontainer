package network

import (
	"fmt"
	"strings"
)

// SetupPortForward creates iptables DNAT rule for port forwarding.
// mapping format: "hostPort:containerPort"
func SetupPortForward(containerIP string, mapping string) error {
	parts := strings.Split(mapping, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid port mapping: %s (use hostPort:containerPort)", mapping)
	}
	hostPort := parts[0]
	containerPort := parts[1]

	// DNAT: redirect incoming traffic on host port to container
	if err := run("iptables", "-t", "nat", "-A", "PREROUTING",
		"-p", "tcp", "--dport", hostPort,
		"-j", "DNAT", "--to-destination", containerIP+":"+containerPort); err != nil {
		return fmt.Errorf("add DNAT rule: %w", err)
	}

	// Also handle traffic from localhost (host to container)
	if err := run("iptables", "-t", "nat", "-A", "OUTPUT",
		"-p", "tcp", "--dport", hostPort, "-j", "DNAT",
		"--to-destination", containerIP+":"+containerPort); err != nil {
		return fmt.Errorf("add OUTPUT DNAT rule: %w", err)
	}

	return nil
}

// RemovePortForward removes the iptables DNAT rules.
func RemovePortForward(containerIP string, mapping string) error {
	parts := strings.Split(mapping, ":")
	if len(parts) != 2 {
		return nil
	}
	hostPort := parts[0]
	containerPort := parts[1]

	// Remove PREROUTING rule
	run("iptables", "-t", "nat", "-D", "PREROUTING",
		"-p", "tcp", "--dport", hostPort,
		"-j", "DNAT", "--to-destination", containerIP+":"+containerPort)

	// Remove OUTPUT rule
	run("iptables", "-t", "nat", "-D", "OUTPUT",
		"-p", "tcp", "--dport", hostPort,
		"-j", "DNAT", "--to-destination", containerIP+":"+containerPort)

	return nil
}
