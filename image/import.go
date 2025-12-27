package image

import "strings"

func ParseImageRef(ref string) (name, tag string) {
	// Split on ":" - if present, we have name:tag
	// If not, name only with default tag "latest"
	parts := strings.SplitN(ref, ":", 2)
	name = parts[0]

	if len(parts) == 2 {
		tag = parts[1]
	} else {
		tag = "latest"
	}

	return name, tag
}
