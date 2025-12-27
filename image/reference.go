package image

import "strings"

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
	result := ImageReference{
		Registry: DefaultRegistry,
		Tag:      "latest",
	}

	// Step 1: Extract tag if present (after last ":")
	// But be careful: registry may have port like "localhost:5000/image"
	if idx := strings.LastIndex(ref, ":"); idx != -1 {
		// Check if ":" is part of tag (no "/" after it) or port (has "/" after)
		afterColon := ref[idx+1:]
		if !strings.Contains(afterColon, "/") {
			result.Tag = afterColon
			ref = ref[:idx]
		}
	}

	// Step 2: Determine if ref contains a registry
	// A registry is present if:
	//   - First component contains "." (e.g., "ghcr.io")
	//   - First component contains ":" (e.g., "localhost:5000")
	//   - First component is "localhost"
	parts := strings.SplitN(ref, "/", 2)

	if len(parts) == 1 {
		// No "/", so it's just an image name like "alpine"
		// Docker Hub official image: add "library/" prefix
		result.Repository = "library/" + parts[0]
	} else {
		firstPart := parts[0]
		hasRegistry := strings.Contains(firstPart, ".") ||
			strings.Contains(firstPart, ":") ||
			firstPart == "localhost"

		if hasRegistry {
			// First part is a registry
			result.Registry = firstPart
			result.Repository = parts[1]
		} else {
			// No registry, just "user/repo" format for Docker Hub
			result.Repository = ref
		}
	}

	return result
}

// String returns the full image reference string.
func (r ImageReference) String() string {
	return r.Registry + "/" + r.Repository + ":" + r.Tag
}
