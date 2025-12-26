package cgroup

// CgroupBasePath is the root cgroup directory for all minicontainer cgroups.
const CgroupBasePath = "/sys/fs/cgroup/minicontainer"

func EnsureParentCgroup() {
	panic("todo")
}

func CreateContainerCgroup(containerID string) {
	panic("todo")
}

func ContainerCgroupPath(containerID string) {
	panic("todo")
}
