package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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

// MountVolume bind-mounts a single volume into the container rootfs.
// Must be called after overlayfs is set up, before pivot_root.
// The containerPath is relative to rootfsPath.
func MountVolume(rootfsPath string, vol VolumeMount) error {
	// Create the mount point inside container rootfs
	targetPath := filepath.Join(rootfsPath, vol.ContainerPath)
	if err := os.MkdirAll(targetPath, 0o755); err != nil {
		return fmt.Errorf("create mount point %s: %w", targetPath, err)
	}

	// Bind mount the host path to container path
	flags := syscall.MS_BIND | syscall.MS_REC
	if err := syscall.Mount(vol.HostPath, targetPath, "", uintptr(flags), ""); err != nil {
		return fmt.Errorf("bind mount %s -> %s: %w", vol.HostPath, targetPath, err)
	}

	// Remount as read-only if requested
	if vol.ReadOnly {
		flags := syscall.MS_BIND | syscall.MS_REC | syscall.MS_RDONLY | syscall.MS_REMOUNT
		if err := syscall.Mount("", targetPath, "", uintptr(flags), ""); err != nil {
			return fmt.Errorf("remount read-only %s: %w", targetPath, err)
		}
	}

	return nil
}

// MountVolumes parses and mounts all volume specifications.
func MountVolumes(rootfsPath string, specs []string) error {
	for _, spec := range specs {
		vol, err := ParseVolumeSpec(spec)
		if err != nil {
			return err
		}
		if err := MountVolume(rootfsPath, vol); err != nil {
			return err
		}
	}
	return nil
}
