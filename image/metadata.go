package image

import "time"

type ImageMetadata struct {
	ID           string    // SHA256 hash of image content (64 hex chars)
	Name         string    // Image name (e.g., "alpine")
	Tag          string    // Image tag (e.g., "latest")
	Layers       []string  // Layer digests in order (bottom to top)
	ConfigDigest string    // Digest of config blob (for registry images, empty for imports)
	CreatedAt    time.Time // When image was created/imported
	Size         int64     // Total size in bytes
}
