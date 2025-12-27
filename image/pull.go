package image

import (
	"fmt"
	"io"
	"os"
)

// Pull downloads an image from a registry and stores it locally.
// Returns the image metadata on success.
func Pull(refStr string) (*ImageMetadata, error) {
	// Step 1: Parse reference
	ref := ParseReference(refStr)
	fmt.Printf("Pulling %s...\n", ref.String())

	// Step 2: Ensure directories exist
	if err := EnsureImageDirs(); err != nil {
		return nil, fmt.Errorf("ensure dirs: %w", err)
	}

	// Step 3: Create registry client and authenticate
	client := NewRegistryClient(ref)
	fmt.Printf("  Authenticating with %s...\n", ref.Registry)
	if err := client.Authenticate(); err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	// Step 4: Fetch manifest
	fmt.Printf("  Fetching manifest...\n")
	manifest, err := client.FetchManifest()
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %w", err)
	}

	// Step 5: Download and extract layers
	var layerDigests []string
	var totalSize int64

	for i, layer := range manifest.Layers {
		fmt.Printf("  Downloading layer %d/%d (%s)...\n", i+1, len(manifest.Layers), layer.Digest[:19])

		// Check if layer already exists (caching)
		if LayerExists(layer.Digest) {
			fmt.Printf("    Layer already exists, skipping.\n")
			layerDigests = append(layerDigests, layer.Digest)
			continue
		}

		// Download layer to temp file
		layerPath, size, err := downloadLayer(client, layer.Digest)
		if err != nil {
			return nil, fmt.Errorf("download layer: %w", err)
		}
		totalSize += size

		// Extract layer
		digest, _, err := ExtractLayer(layerPath)
		if err != nil {
			os.Remove(layerPath)
			return nil, fmt.Errorf("extract layer: %w", err)
		}
		os.Remove(layerPath) // Clean up temp file

		layerDigests = append(layerDigests, digest)
	}

	// Step 6: Create and save metadata
	meta := &ImageMetadata{
		ID:           manifest.Config.Digest[7:], // Strip "sha256:" prefix
		Name:         ref.Repository,
		Tag:          ref.Tag,
		Layers:       layerDigests,
		ConfigDigest: manifest.Config.Digest,
		CreatedAt:    time.Now(),
		Size:         totalSize,
	}

	// Use simple name for storage (without registry prefix)
	name, _ := ParseImageRef(refStr)
	meta.Name = name

	if err := SaveMetadata(meta); err != nil {
		return nil, fmt.Errorf("save metadata: %w", err)
	}

	fmt.Printf("  Done! Image %s:%s pulled.\n", meta.Name, meta.Tag)
	return meta, nil
}

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
