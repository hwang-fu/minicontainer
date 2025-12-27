package image

import (
	"fmt"
	"strings"
	"time"
)

// ParseImageRef splits an image reference into name and tag.
// If no tag is provided, defaults to "latest".
//
// Examples:
//   - "alpine" -> ("alpine", "latest")
//   - "alpine:3.19" -> ("alpine", "3.19")
//   - "myapp:v1.0" -> ("myapp", "v1.0")
//
// Parameters:
//   - ref: image reference string in "name" or "name:tag" format
//
// Returns:
//   - name: the image name
//   - tag: the image tag (defaults to "latest" if not specified)
func ParseImageRef(ref string) (name, tag string) {
	// Split on ":" - if present, we have name:tag
	// If not, name only with default tag "latest"
	parts := strings.SplitN(ref, ":", 2)
	name = parts[0]

	if len(parts) == 2 {
		tag = parts[1]
	} else {
		tag = "latest"
	}

	return name, tag
}

// ImportTarball imports a rootfs tarball as a single-layer image.
// This creates an image that can be used with `minicontainer run <name:tag>`.
//
// The process:
//  1. Ensure base directories exist
//  2. Extract the tarball to a layer directory (content-addressable)
//  3. Create image metadata with the layer digest
//  4. Save metadata to the image directory
//
// Parameters:
//   - tarballPath: path to the .tar or .tar.gz rootfs archive
//   - ref: image reference in "name" or "name:tag" format
//
// Returns:
//   - *ImageMetadata: the created image metadata
//   - error: any error during import
func ImportTarball(tarballPath, ref string) (*ImageMetadata, error) {
	// Step 1: Ensure base directories exist
	if err := EnsureImageDirs(); err != nil {
		return nil, fmt.Errorf("ensure image dirs: %w", err)
	}

	// Step 2: Parse the image reference into name and tag
	name, tag := ParseImageRef(ref)

	// Step 3: Extract the tarball to a content-addressable layer directory
	// ExtractLayer returns the digest (used as layer ID) and size
	digest, size, err := ExtractLayer(tarballPath)
	if err != nil {
		return nil, fmt.Errorf("extract layer: %w", err)
	}

	// Step 4: Create image metadata
	// For imported tarballs, we use the layer digest as the image ID
	// (since there's only one layer, its digest uniquely identifies the image)
	meta := &ImageMetadata{
		ID:        strings.TrimPrefix(digest, "sha256:"), // Store just the hex part
		Name:      name,
		Tag:       tag,
		Layers:    []string{digest}, // Single layer for imported tarball
		CreatedAt: time.Now(),
		Size:      size,
		// ConfigDigest is empty for imported images (no OCI config)
	}

	// Step 5: Save metadata to disk
	if err := SaveMetadata(meta); err != nil {
		return nil, fmt.Errorf("save metadata: %w", err)
	}

	return meta, nil
}
