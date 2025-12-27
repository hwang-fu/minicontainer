package image

import (
	"fmt"
	"slices"
)

func RemoveImage(ref string) error {
	// Try to find the image by name:tag first
	name, tag := ParseImageRef(ref)
	meta, err := LoadMetadata(name, tag)
	if err != nil {
		// Not found by name:tag, try to find by ID
		meta, err = findImageByID(ref)
		if err != nil {
			return fmt.Errorf("image %s not found", ref)
		}
	}

	// Collect layers to potentially remove
	layersToCheck := meta.Layers

	// Remove image metadata directory
	imageDir := ImageDir(meta.Name, meta.Tag)
	if err := os.RemoveAll(imageDir); err != nil {
		return fmt.Errorf("remove image metadata: %w", err)
	}

	// Clean up empty parent directory (name directory) if no more tags
	nameDir := filepath.Dir(imageDir)
	entries, _ := os.ReadDir(nameDir)
	if len(entries) == 0 {
		os.Remove(nameDir) // Best effort, ignore errors
	}

	// Remove layers that are no longer referenced by any image
	for _, layerDigest := range layersToCheck {
		if !isLayerReferenced(layerDigest) {
			RemoveLayer(layerDigest)
		}
	}

	return nil
}

// findImageByID searches for an image by full or short ID.
// Returns the image metadata if found.
func findImageByID(id string) (*ImageMetadata, error) {
	images, err := ListImages()
	if err != nil {
		return nil, err
	}

	for _, img := range images {
		// Match full ID or prefix (short ID)
		if img.ID == id || (len(id) >= 4 && len(img.ID) >= len(id) && img.ID[:len(id)] == id) {
			return img, nil
		}
	}

	return nil, fmt.Errorf("image not found")
}

// isLayerReferenced checks if any image references this layer.
// Used to determine if a layer can be safely deleted.
func isLayerReferenced(layerDigest string) bool {
	images, err := ListImages()
	if err != nil {
		return true // Assume referenced on error (safer)
	}

	return slices.ContainsFunc(images, func(img *ImageMetadata) bool {
		return slices.Contains(img.Layers, layerDigest)
	})
}
