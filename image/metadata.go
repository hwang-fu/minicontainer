package image

import "time"

type ImageMetadata struct {
	ID           string    `json:"id"`            // SHA256 hash of image content (64 hex chars)
	Name         string    `json:"name"`          // Image name (e.g., "alpine")
	Tag          string    `json:"tag"`           // Image tag (e.g., "latest")
	Layers       []string  `json:"layers"`        // Layer digests in order (bottom to top)
	ConfigDigest string    `json:"config_digest"` // Digest of config blob (for registry images, empty for imports)
	CreatedAt    time.Time `json:"created_at"`    // When image was created/imported
	Size         int64     `json:"size"`          // Total size in bytes
}
