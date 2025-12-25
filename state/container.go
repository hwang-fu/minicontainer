package state

// ContainerStatus represents the lifecycle state of a container.
type ContainerStatus string

const (
	StatusCreated ContainerStatus = "created"
	StatusRunning ContainerStatus = "running"
	StatusStopped ContainerStatus = "stopped"
)
