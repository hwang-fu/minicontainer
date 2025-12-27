package image

import "fmt"

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
