package network

import (
	"fmt"
	"sync"
)

var (
	// Simple IPAM: track allocated IPs
	allocatedIPs = make(map[string]bool)
	ipamMutex    sync.Mutex
	nextIP       = 2 // Start at 172.17.0.2
)

// AllocateIP returns the next available IP in 172.17.0.0/16.
func AllocateIP() (string, error) {
	ipamMutex.Lock()
	defer ipamMutex.Unlock()

	for nextIP < 65534 { // Max IPs in /16
		ip := fmt.Sprintf("172.17.0.%d", nextIP)
		nextIP++
		if !allocatedIPs[ip] {
			allocatedIPs[ip] = true
			return ip, nil
		}
	}
	return "", fmt.Errorf("no available IPs")
}

// ReleaseIP marks an IP as available.
func ReleaseIP(ip string) {
	ipamMutex.Lock()
	defer ipamMutex.Unlock()
	delete(allocatedIPs, ip)
}

// Gateway returns the bridge IP.
func Gateway() string {
	return "172.17.0.1"
}
