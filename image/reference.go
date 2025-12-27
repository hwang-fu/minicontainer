package image

// ImageReference represents a fully qualified image reference.
// Example: docker.io/library/alpine:3.19
type ImageReference struct {
	Registry   string // e.g., "registry-1.docker.io"
	Repository string // e.g., "library/alpine"
	Tag        string // e.g., "3.19" (default: "latest")
}
