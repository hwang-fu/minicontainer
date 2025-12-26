package network

import "sync"

var (
	// Simple IPAM: track allocated IPs
	allocatedIPs = make(map[string]bool)
	ipamMutex    sync.Mutex
	nextIP       = 2 // Start at 172.17.0.2
)
