package image

// ImageReference represents a fully qualified image reference.
// Example: docker.io/library/alpine:3.19
type ImageReference struct {
	Registry   string // e.g., "registry-1.docker.io"
	Repository string // e.g., "library/alpine"
	Tag        string // e.g., "3.19" (default: "latest")
}

// DefaultRegistry is the Docker Hub registry endpoint.
const DefaultRegistry = "registry-1.docker.io"

// ParseReference parses an image reference string into its components.
// Handles various formats:
//   - "alpine"                    -> registry-1.docker.io/library/alpine:latest
//   - "alpine:3.19"               -> registry-1.docker.io/library/alpine:3.19
//   - "nginx"                     -> registry-1.docker.io/library/nginx:latest
//   - "myuser/myapp"              -> registry-1.docker.io/myuser/myapp:latest
//   - "ghcr.io/owner/repo:v1"     -> ghcr.io/owner/repo:v1
//
// Docker Hub special cases:
//   - Official images (no /) get "library/" prefix
//   - Docker Hub registry is "registry-1.docker.io"
func ParseReference(ref string) ImageReference {
	// TODO: implement
	panic("todo")
}

// String returns the full image reference string.
func (r ImageReference) String() string {
	return r.Registry + "/" + r.Repository + ":" + r.Tag
}
