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
	randBytes := make([]byte, 32)
	if _, err := rand.Read(randBytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	hash := sha256.Sum256(randBytes)
	return hex.EncodeToString(hash[:]), nil
}
