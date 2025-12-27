package image

// LookupImage finds an image by reference and returns the path to its root layer.
// For single-layer images (imports), this returns the layer directory directly.
// For multi-layer images (pulled), this would need to set up overlayfs (future).
//
// Parameters:
//   - ref: image reference in "name" or "name:tag" format
//
// Returns:
//   - rootfsPath: path to the layer directory to use as container rootfs
//   - error: if image not found or has no layers
func LookupImage(ref string) (rootfsPath string, err error) {
	// Parse the image reference
	name, tag := ParseImageRef(ref)

	// Load image metadata
	meta, err := LoadMetadata(name, tag)
	if err != nil {
		// Check if it's a "not found" error
		if os.IsNotExist(err) {
			return "", fmt.Errorf("image %s:%s not found", name, tag)
		}
		return "", fmt.Errorf("load image metadata: %w", err)
	}

	// Verify image has at least one layer
	if len(meta.Layers) == 0 {
		return "", fmt.Errorf("image %s:%s has no layers", name, tag)
	}

	// For now, use the first (and typically only) layer as rootfs
	// Multi-layer support would require stacking layers with overlayfs
	rootfsPath = LayerDir(meta.Layers[0])

	// Verify the layer exists
	if !LayerExists(meta.Layers[0]) {
		return "", fmt.Errorf("layer %s not found for image %s:%s", meta.Layers[0][:12], name, tag)
	}

	return rootfsPath, nil
}
