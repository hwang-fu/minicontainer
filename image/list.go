package image

// ListImages returns metadata for all locally stored images.
// Scans the image directory structure: /var/lib/minicontainer/images/<name>/<tag>/
//
// Returns:
//   - slice of ImageMetadata for all found images
//   - error if directory cannot be read
func ListImages() ([]*ImageMetadata, error) {
	var images []*ImageMetadata

	// Check if base directory exists
	if _, err := os.Stat(ImageBaseDir); os.IsNotExist(err) {
		// No images directory yet - return empty list
		return images, nil
	}

	// Walk the images directory: images/<name>/<tag>/manifest.json
	// First level: image names
	names, err := os.ReadDir(ImageBaseDir)
	if err != nil {
		return nil, err
	}

	for _, nameEntry := range names {
		if !nameEntry.IsDir() {
			continue
		}
		namePath := filepath.Join(ImageBaseDir, nameEntry.Name())

		// Second level: tags for this image name
		tags, err := os.ReadDir(namePath)
		if err != nil {
			continue // Skip unreadable directories
		}

		for _, tagEntry := range tags {
			if !tagEntry.IsDir() {
				continue
			}

			// Load metadata for this name:tag
			meta, err := LoadMetadata(nameEntry.Name(), tagEntry.Name())
			if err != nil {
				continue // Skip images with invalid/missing metadata
			}

			images = append(images, meta)
		}
	}

	return images, nil
}
