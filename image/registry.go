package image

import "net/http"

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
