package image

import (
	"fmt"
	"io"
	"os"
)

// downloadLayer downloads a layer blob to a temp file.
// Returns the path to the temp file and its size.
func downloadLayer(client *RegistryClient, digest string) (string, int64, error) {
	body, size, err := client.FetchBlob(digest)
	if err != nil {
		return "", 0, err
	}
	defer body.Close()

	// Create temp file for layer tarball
	tmpFile, err := os.CreateTemp("", "layer-*.tar.gz")
	if err != nil {
		return "", 0, fmt.Errorf("create temp file: %w", err)
	}
	defer tmpFile.Close()

	// Copy blob to temp file
	written, err := io.Copy(tmpFile, body)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", 0, fmt.Errorf("download layer: %w", err)
	}

	if size > 0 && written != size {
		os.Remove(tmpFile.Name())
		return "", 0, fmt.Errorf("size mismatch: expected %d, got %d", size, written)
	}

	return tmpFile.Name(), written, nil
}
