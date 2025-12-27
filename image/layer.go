package image

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	// Step 1: Compute digest of the tarball file
	// This gives us the content-addressable name for the layer
	digest, err = computeDigest(tarballPath)
	if err != nil {
		return "", 0, fmt.Errorf("compute layer digest: %w", err)
	}

	// Step 2: Check if this layer is already cached
	// Content-addressable storage means identical content = identical digest
	if LayerExists(digest) {
		// Layer already extracted, get its size and return
		size, err = dirSize(LayerDir(digest))
		if err != nil {
			return "", 0, fmt.Errorf("get cached layer size: %w", err)
		}
		return digest, size, nil
	}

	// Step 3: Create the layer directory
	layerPath := LayerDir(digest)
	if err := os.MkdirAll(layerPath, 0o755); err != nil {
		return "", 0, fmt.Errorf("create layer dir: %w", err)
	}

	// Step 4: Extract tarball to layer directory
	// Use tar command for simplicity - handles .tar and .tar.gz automatically
	size, err = extractTarball(tarballPath, layerPath)
	if err != nil {
		// Clean up partial extraction on failure
		os.RemoveAll(layerPath)
		return "", 0, fmt.Errorf("extract tarball: %w", err)
	}

	return digest, size, nil
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
// Uses os.RemoveAll to recursively delete the directory and all contents.
//
// Parameters:
//   - digest: the "sha256:<hex>" identifier of the layer to remove
//
// Returns:
//   - error if removal fails (nil if layer doesn't exist - idempotent)
func RemoveLayer(digest string) error {
	// Get the path where this layer is stored
	path := LayerDir(digest)

	// RemoveAll deletes path and any children it contains.
	// Returns nil if path doesn't exist (idempotent operation).
	// This is safe because LayerDir always returns a path under LayerBaseDir.
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove layer %s: %w", digest, err)
	}

	return nil
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

// dirSize calculates the total size of all files in a directory tree.
// Used to report layer size after extraction.
//
// Parameters:
//   - path: root directory to calculate size for
//
// Returns:
//   - total size in bytes of all regular files
func dirSize(path string) (int64, error) {
	var size int64

	// Walk the directory tree, summing file sizes
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Only count regular files (not directories, symlinks, etc.)
		if info.Mode().IsRegular() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// extractTarball extracts a tar archive to the destination directory.
// Supports both .tar and .tar.gz/.tgz files (auto-detected by tar command).
// Uses the system tar command for simplicity and broad format support.
//
// Parameters:
//   - tarballPath: path to the .tar or .tar.gz file
//   - destDir: directory to extract contents into (must exist)
//
// Returns:
//   - size: total bytes of extracted files
//   - error: any error during extraction
func extractTarball(tarballPath, destDir string) (int64, error) {
	// Use system tar command with auto-compression detection (-a flag)
	// -x: extract
	// -f: read from file
	// -C: change to directory before extracting
	//
	// Note: GNU tar auto-detects gzip compression, so we don't need -z flag
	cmd := exec.Command("tar", "-xf", tarballPath, "-C", destDir)

	// Capture stderr for error messages
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run the extraction
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("tar extract failed: %v: %s", err, stderr.String())
	}

	// Calculate total size of extracted files
	size, err := dirSize(destDir)
	if err != nil {
		return 0, fmt.Errorf("calculate extracted size: %w", err)
	}

	return size, nil
}
