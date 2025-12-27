package image

import "net/http"

// RegistryClient handles communication with OCI registries.
type RegistryClient struct {
	ref    ImageReference
	token  string       // Bearer token for authentication
	client *http.Client // HTTP client for requests
}
