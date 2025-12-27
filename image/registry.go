package image

import (
	"net/http"
	"strings"
)

// RegistryClient handles communication with OCI registries.
type RegistryClient struct {
	ref    ImageReference
	token  string       // Bearer token for authentication
	client *http.Client // HTTP client for requests
}

// NewRegistryClient creates a client for the given image reference.
func NewRegistryClient(ref ImageReference) *RegistryClient {
	return &RegistryClient{
		ref:    ref,
		client: &http.Client{},
	}
}

// parseAuthHeader extracts realm and service from WWW-Authenticate header.
// Example: Bearer realm="https://auth.docker.io/token",service="registry.docker.io"
func parseAuthHeader(header string) (realm, service string) {
	// Remove "Bearer " prefix
	header = strings.TrimPrefix(header, "Bearer ")

	// Parse key="value" pairs
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "realm=") {
			realm = strings.Trim(strings.TrimPrefix(part, "realm="), "\"")
		} else if strings.HasPrefix(part, "service=") {
			service = strings.Trim(strings.TrimPrefix(part, "service="), "\"")
		}
	}
	return realm, service
}

// doRequest makes an authenticated request to the registry.
func (c *RegistryClient) doRequest(method, url string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	// Add auth token if we have one
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return c.client.Do(req)
}
