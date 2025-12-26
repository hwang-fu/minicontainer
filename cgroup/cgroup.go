package cgroup

import "path/filepath"

// CgroupBasePath is the root cgroup directory for all minicontainer cgroups.
const CgroupBasePath = "/sys/fs/cgroup/minicontainer"

// ContainerCgroupPath returns the cgroup directory path for a container.
func ContainerCgroupPath(containerID string) string {
	return filepath.Join(CgroupBasePath, containerID)
}

func EnsureParentCgroup() {
	panic("todo")
}

func CreateContainerCgroup(containerID string) {
	panic("todo")
}
