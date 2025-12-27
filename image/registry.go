package image

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ManifestV2 represents an OCI/Docker image manifest (schema v2).
// Contains references to the config and layer blobs.
type ManifestV2 struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int64  `json:"size"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int64  `json:"size"`
	} `json:"layers"`
}

// ManifestList represents a multi-architecture manifest list.
// Docker Hub returns this for multi-arch images like "alpine".
type ManifestList struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Manifests     []struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int64  `json:"size"`
		Platform  struct {
			Architecture string `json:"architecture"`
			OS           string `json:"os"`
		} `json:"platform"`
	} `json:"manifests"`
}

// ImageConfig represents the OCI image configuration.
// Contains runtime settings like Env, Cmd, Entrypoint.
type ImageConfig struct {
	Config struct {
		Env        []string `json:"Env"`
		Cmd        []string `json:"Cmd"`
		Entrypoint []string `json:"Entrypoint"`
		WorkingDir string   `json:"WorkingDir"`
		User       string   `json:"User"`
	} `json:"config"`
}

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

// Authenticate obtains a bearer token for the registry.
// For Docker Hub, this involves:
//  1. Request to registry returns 401 with WWW-Authenticate header
//  2. Parse header to get realm, service, scope
//  3. Request token from auth endpoint (anonymous for public images)
func (c *RegistryClient) Authenticate() error {
	// Step 1: Make initial request to trigger 401
	url := fmt.Sprintf("https://%s/v2/", c.ref.Registry)
	resp, err := c.client.Get(url)
	if err != nil {
		return fmt.Errorf("registry ping: %w", err)
	}
	defer resp.Body.Close()

	// If 200, no auth needed
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	// Expect 401 with WWW-Authenticate header
	if resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Step 2: Parse WWW-Authenticate header
	// Format: Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:library/alpine:pull"
	authHeader := resp.Header.Get("WWW-Authenticate")
	if authHeader == "" {
		return fmt.Errorf("missing WWW-Authenticate header")
	}

	realm, service := parseAuthHeader(authHeader)
	if realm == "" {
		return fmt.Errorf("could not parse auth header: %s", authHeader)
	}

	// Step 3: Request token
	scope := fmt.Sprintf("repository:%s:pull", c.ref.Repository)
	tokenURL := fmt.Sprintf("%s?service=%s&scope=%s", realm, service, scope)

	tokenResp, err := c.client.Get(tokenURL)
	if err != nil {
		return fmt.Errorf("token request: %w", err)
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(tokenResp.Body)
		return fmt.Errorf("token request failed: %d: %s", tokenResp.StatusCode, body)
	}

	// Parse token response
	var tokenData struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		return fmt.Errorf("parse token response: %w", err)
	}

	c.token = tokenData.Token
	return nil
}

// FetchManifest retrieves the image manifest from the registry.
// The manifest contains:
//   - Config digest (image configuration with Env, Cmd, etc.)
//   - Layer digests (filesystem layers to download)
//
// Uses Docker manifest v2 schema 2 or OCI manifest format.
func (c *RegistryClient) FetchManifest() (*ManifestV2, error) {
	// Build manifest URL: /v2/<repo>/manifests/<tag>
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s",
		c.ref.Registry, c.ref.Repository, c.ref.Tag)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create manifest request: %w", err)
	}

	// Add auth token
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	// Accept both Docker and OCI manifest formats
	// Docker v2 schema 2 is most common for Docker Hub
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.oci.image.manifest.v1+json",
	}, ", "))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("manifest request failed: %d: %s", resp.StatusCode, body)
	}

	var manifest ManifestV2
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	return &manifest, nil
}

// FetchBlob downloads a blob (layer or config) by digest.
// Returns the blob content as a reader. Caller must close it.
//
// Parameters:
//   - digest: the "sha256:..." digest of the blob
//
// Returns:
//   - io.ReadCloser: blob content stream
//   - int64: content length
//   - error: any error during fetch
func (c *RegistryClient) FetchBlob(digest string) (io.ReadCloser, int64, error) {
	// Build blob URL: /v2/<repo>/blobs/<digest>
	url := fmt.Sprintf("https://%s/v2/%s/blobs/%s",
		c.ref.Registry, c.ref.Repository, digest)

	resp, err := c.doRequest("GET", url)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch blob: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, 0, fmt.Errorf("blob request failed: %d: %s", resp.StatusCode, body)
	}

	return resp.Body, resp.ContentLength, nil
}

// FetchConfig downloads and parses the image configuration.
// The config contains runtime settings (Env, Cmd, Entrypoint, etc.)
func (c *RegistryClient) FetchConfig(digest string) (*ImageConfig, error) {
	body, _, err := c.FetchBlob(digest)
	if err != nil {
		return nil, fmt.Errorf("fetch config blob: %w", err)
	}
	defer body.Close()

	var config ImageConfig
	if err := json.NewDecoder(body).Decode(&config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &config, nil
}

// parseAuthHeader extracts realm and service from WWW-Authenticate header.
// Example: Bearer realm="https://auth.docker.io/token",service="registry.docker.io"
func parseAuthHeader(header string) (realm, service string) {
	// Remove "Bearer " prefix
	header = strings.TrimPrefix(header, "Bearer ")

	// Parse key="value" pairs
	for part := range strings.SplitSeq(header, ",") {
		part = strings.TrimSpace(part)
		if val, ok := strings.CutPrefix(part, "realm="); ok {
			realm = strings.Trim(val, "\"")
		} else if val, ok := strings.CutPrefix(part, "service="); ok {
			service = strings.Trim(val, "\"")
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
