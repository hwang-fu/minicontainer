package image

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// LayerDir returns the path where a layer's contents are extracted.
// Example: LayerDir("sha256:abc123...") -> "/var/lib/minicontainer/layers/abc123..."
func LayerDir(digest string) string {
	// Strip the algorithm prefix (sha256:) to avoid colons in paths
	cleanDigest := strings.TrimPrefix(digest, "sha256:")
	return filepath.Join(LayerBaseDir, cleanDigest)
}

// EnsureImageDirs creates the base image and layer directories if they don't exist.
func EnsureImageDirs() error {
	if err := os.MkdirAll(ImageBaseDir, 0o755); err != nil {
		return fmt.Errorf("create image dir: %w", err)
	}
	if err := os.MkdirAll(LayerBaseDir, 0o755); err != nil {
		return fmt.Errorf("create layer dir: %w", err)
	}
	return nil
}
