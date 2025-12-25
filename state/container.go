package state

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

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

// ContainerDir returns the directory for a container's data.
func ContainerDir(containerID string) string {
	return StateBaseDir + "/" + containerID
}

// SaveState writes the container state to disk as JSON.
func SaveState(cs *ContainerState) error {
	dir := ContainerDir(cs.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create container dir: %w", err)
	}

	data, err := json.MarshalIndent(cs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	path := StatePath(cs.ID)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}
	return nil
}

// NewContainerState creates a new state with initial values.
func NewContainerState(id, name, rootfsPath string, command []string) *ContainerState {
	return &ContainerState{
		ID:         id,
		Name:       name,
		Command:    command,
		Status:     StatusCreated,
		CreatedAt:  time.Now(),
		RootfsPath: rootfsPath,
	}
}

// LoadState reads a container's state from disk.
func LoadState(containerID string) (*ContainerState, error) {
	path := StatePath(containerID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}

	var cs ContainerState
	if err := json.Unmarshal(data, &cs); err != nil {
		return nil, fmt.Errorf("unmarshal state: %w", err)
	}
	return &cs, nil
}
