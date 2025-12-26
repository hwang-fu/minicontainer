package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// CgroupBasePath is the root cgroup directory for all minicontainer cgroups.
const CgroupBasePath = "/sys/fs/cgroup/minicontainer"

// ContainerCgroupPath returns the cgroup directory path for a container.
func ContainerCgroupPath(containerID string) string {
	return filepath.Join(CgroupBasePath, containerID)
}

// EnsureParentCgroup creates the minicontainer parent cgroup and enables controllers.
// This MUST be called before creating any container cgroups.
// Enables: cpu, memory, pids controllers for child cgroups.
func EnsureParentCgroup() error {
	// Create parent cgroup directory
	if err := os.MkdirAll(CgroupBasePath, 0o755); err != nil {
		return fmt.Errorf("create cgroup dir: %w", err)
	}

	// Enable controllers for child cgroups by writing to subtree_control
	// Format: "+cpu +memory +pids" enables these controllers for children
	subtreeControlPath := filepath.Join(CgroupBasePath, "cgroup.subtree_control")
	if err := os.WriteFile(subtreeControlPath, []byte("+cpu +memory +pids"), 0o644); err != nil {
		return fmt.Errorf("enable cgroup controllers: %w", err)
	}

	return nil
}

// CreateContainerCgroup creates a cgroup directory for a specific container.
// Returns the path to the created cgroup directory.
func CreateContainerCgroup(containerID string) (string, error) {
	// Ensure parent exists and has controllers enabled
	if err := EnsureParentCgroup(); err != nil {
		return "", err
	}

	// Create container-specific cgroup directory
	cgroupPath := ContainerCgroupPath(containerID)
	if err := os.MkdirAll(cgroupPath, 0o755); err != nil {
		return "", fmt.Errorf("create container cgroup: %w", err)
	}

	return cgroupPath, nil
}

// AddProcessToCgroup writes a PID to the cgroup's cgroup.procs file.
// This adds the process (and its children) to the cgroup.
func AddProcessToCgroup(cgroupPath string, pid int) error {
	procsPath := filepath.Join(cgroupPath, "cgroup.procs")
	if err := os.WriteFile(procsPath, strconv.AppendInt(nil, int64(pid), 10), 0o644); err != nil {
		return fmt.Errorf("add process to cgroup: %w", err)
	}
	return nil
}
