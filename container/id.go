package container

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateContainerID creates a unique 64-character hex container ID.
// Uses crypto/rand for randomness and SHA256 for the hash.
func GenerateContainerID() (string, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	hash := sha256.Sum256(randomBytes)
	return hex.EncodeToString(hash[:]), nil
}

// ShortID returns the first 12 characters of a container ID.
// Used for display in ps, logs, and user-facing output.
func ShortID(fullID string) string {
	if len(fullID) < 12 {
		return fullID
	}
	return fullID[:12]
}
