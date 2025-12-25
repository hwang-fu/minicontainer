package container

// GenerateContainerID creates a unique 64-character hex container ID.
// Uses crypto/rand for randomness and SHA256 for the hash.
func GenerateContainerID() (string, error) {
	panic("todo")
}

// ShortID returns the first 12 characters of a container ID.
// Used for display in ps, logs, and user-facing output.
func ShortID(fullID string) string {
	if len(fullID) < 12 {
		return fullID
	}
	return fullID[:12]
}
