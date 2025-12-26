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

// SetMemoryLimit writes the memory limit to the cgroup.
// limitBytes is the memory limit in bytes.
func SetMemoryLimit(cgroupPath string, limitBytes int64) error {
	memoryMaxPath := filepath.Join(cgroupPath, "memory.max")
	if err := os.WriteFile(memoryMaxPath, []byte(strconv.FormatInt(limitBytes, 10)), 0o644); err != nil {
		return fmt.Errorf("set memory limit: %w", err)
	}
	return nil
}

// ParseMemoryLimit converts human-readable memory string to bytes.
// Supports: k/K (kilobytes), m/M (megabytes), g/G (gigabytes)
// Examples: "256m" -> 268435456, "1g" -> 1073741824
func ParseMemoryLimit(limit string) (int64, error) {
	if limit == "" {
		return 0, nil
	}

	limit = strings.TrimSpace(limit)
	unit := strings.ToLower(limit[len(limit)-1:])
	valueStr := limit[:len(limit)-1]
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		// No unit suffix, try parsing whole string as bytes
		return strconv.ParseInt(limit, 10, 64)
	}

	switch unit {
	case "k":
		return value * 1024, nil
	case "m":
		return value * 1024 * 1024, nil
	case "g":
		return value * 1024 * 1024 * 1024, nil
	default:
		return strconv.ParseInt(limit, 10, 64)
	}
}
