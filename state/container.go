package state

import "time"

// ContainerStatus represents the lifecycle state of a container.
type ContainerStatus string

const (
	StatusCreated ContainerStatus = "created"
	StatusRunning ContainerStatus = "running"
	StatusStopped ContainerStatus = "stopped"
)

// ContainerState holds all persistent metadata for a container.
// Serialized to JSON at /var/lib/minicontainer/containers/<id>/state.json
type ContainerState struct {
	ID         string          `json:"id"`          // Full 64-char container ID
	Name       string          `json:"name"`        // User-provided or short ID
	Command    []string        `json:"command"`     // Command and arguments
	Status     ContainerStatus `json:"status"`      // created, running, stopped
	PID        int             `json:"pid"`         // Host PID of container init process
	CreatedAt  time.Time       `json:"created_at"`  // When container was created
	ExitCode   int             `json:"exit_code"`   // Exit code (valid when stopped)
	RootfsPath string          `json:"rootfs_path"` // Path to container rootfs
}

// StateBaseDir returns the base directory for all container state.
const StateBaseDir = "/var/lib/minicontainer/containers"

// StatePath returns the path to a container's state.json file.
func StatePath(containerID string) string {
	return StateBaseDir + "/" + containerID + "/state.json"
}
