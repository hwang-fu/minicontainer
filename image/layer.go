package image

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// ExtractLayer extracts a tarball to the layer directory and returns its digest.
// The digest is computed from the tarball content (SHA256) and used as the
// directory name for content-addressable storage.
//
// Parameters:
//   - tarballPath: path to the .tar or .tar.gz file to extract
//
// Returns:
//   - digest: the "sha256:<hex>" content hash identifying this layer
//   - size: total bytes of the extracted layer
//   - error: any error during extraction
//
// The layer is stored at: /var/lib/minicontainer/layers/sha256:<hash>/
func ExtractLayer(tarballPath string) (digest string, size int64, err error) {
	// TODO: implement
	panic("todo")
}

// LayerExists checks if a layer with the given digest already exists.
// Used to skip re-extraction of cached layers.
//
// Parameters:
//   - digest: the "sha256:<hex>" identifier to check
//
// Returns:
//   - true if layer directory exists and is non-empty
func LayerExists(digest string) bool {
	// Get the path where this layer would be stored
	path := LayerDir(digest)

	// Stat the path to check if it exists
	// os.Stat follows symlinks; returns error if path doesn't exist
	info, err := os.Stat(path)
	if err != nil {
		// Path doesn't exist or isn't accessible
		return false
	}

	// Verify it's actually a directory, not a file
	return info.IsDir()
}

// RemoveLayer deletes a layer directory by its digest.
// Called during image removal when layer is no longer referenced.
//
// Parameters:
//   - digest: the "sha256:<hex>" identifier of the layer to remove
func RemoveLayer(digest string) error {
	// TODO: implement
	panic("todo")
}

// computeDigest calculates the SHA256 hash of a file.
// Returns the digest in "sha256:<hex>" format, matching OCI content-addressable
// storage conventions (e.g., "sha256:a3ed95caeb02...").
//
// Parameters:
//   - filePath: path to the file to hash
//
// Returns:
//   - digest string in "sha256:<64-hex-chars>" format
//   - error if file cannot be read
func computeDigest(filePath string) (string, error) {
	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file for digest: %w", err)
	}
	defer file.Close()

	// Create a new SHA256 hasher
	// SHA256 is the standard hash algorithm for OCI image digests
	hasher := sha256.New()

	// Copy file contents through the hasher
	// io.Copy streams the file in chunks, avoiding loading entire file into memory
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("hash file: %w", err)
	}

	// Format as "sha256:<hex>" - the OCI digest format
	// hasher.Sum(nil) returns the final hash as []byte
	// %x formats bytes as lowercase hexadecimal
	return fmt.Sprintf("sha256:%x", hasher.Sum(nil)), nil
}
