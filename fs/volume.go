package fs

// VolumeMount represents a bind mount from host to container.
type VolumeMount struct {
	HostPath      string // Path on the host
	ContainerPath string // Path inside the container
	ReadOnly      bool   // If true, mount as read-only
}
