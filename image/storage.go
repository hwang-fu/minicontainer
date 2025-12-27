package image

import (
	"os"
	"path/filepath"
)

const (
	ImageBaseDir = "/var/lib/minicontainer/images"
	LayerBaseDir = "/var/lib/minicontainer/layers"
)

// ImageDir returns the path to store an image's metadata.
// Example: ImageDir("alpine", "latest") -> "/var/lib/minicontainer/images/alpine/latest"
func ImageDir(name, tag string) string {
	return filepath.Join(ImageBaseDir, name, tag)
}
