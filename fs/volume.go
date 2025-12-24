package fs

import (
	"fmt"
	"strings"
)

// VolumeMount represents a bind mount from host to container.
type VolumeMount struct {
	HostPath      string // Path on the host
	ContainerPath string // Path inside the container
	ReadOnly      bool   // If true, mount as read-only
}

// ParseVolumeSpec parses a volume specification string.
// Format: "host:container" or "host:container:ro"
// Returns the parsed VolumeMount or an error.
func ParseVolumeSpec(spec string) (VolumeMount, error) {
	parts := strings.Split(spec, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return VolumeMount{}, fmt.Errorf("invalid volume spec %q: expected host:container[:ro]", spec)
	}

	vol := VolumeMount{
		HostPath:      parts[0],
		ContainerPath: parts[1],
	}

	if len(parts) == 3 {
		if parts[2] == "ro" {
			vol.ReadOnly = true
		} else {
			return VolumeMount{}, fmt.Errorf("invalid volume option %q: expected 'ro'", parts[2])
		}
	}

	return vol, nil
}
