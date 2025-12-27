package image

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type ImageMetadata struct {
	ID           string    `json:"id"`            // SHA256 hash of image content (64 hex chars)
	Name         string    `json:"name"`          // Image name (e.g., "alpine")
	Tag          string    `json:"tag"`           // Image tag (e.g., "latest")
	Layers       []string  `json:"layers"`        // Layer digests in order (bottom to top)
	ConfigDigest string    `json:"config_digest"` // Digest of config blob (for registry images, empty for imports)
	CreatedAt    time.Time `json:"created_at"`    // When image was created/imported
	Size         int64     `json:"size"`          // Total size in bytes
}

// SaveMetadata writes image metadata to manifest.json in the image directory.
func SaveMetadata(meta *ImageMetadata) error {
	dir := ImageDir(meta.Name, meta.Tag)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create image dir: %w", err)
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

// LoadMetadata reads image metadata from manifest.json.
func LoadMetadata(name, tag string) (*ImageMetadata, error) {
	path := filepath.Join(ImageDir(name, tag), "manifest.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var meta ImageMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	return &meta, nil
}
