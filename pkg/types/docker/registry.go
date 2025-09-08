package docker

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"runtime"
	"strings"
)

// DockerRegistryClient provides shared functionality for interacting with Docker registries
type DockerRegistryClient struct {
	token       string
	dockerImage *DockerImage
}

func NewDockerRegistryClient(dockerImage *DockerImage) *DockerRegistryClient {
	return &DockerRegistryClient{
		dockerImage: dockerImage,
	}
}

func (drc *DockerRegistryClient) RefreshToken() error {
	imageName, tag := drc.ParseImageName(drc.dockerImage.Image)

	// Reset the token so we know if a refresh is needed versus auth failure
	drc.token = ""

	// Always try to access the registry first to get the WWW-Authenticate header or see if a token is required.
	registryBase := drc.getRegistryBase()
	url := fmt.Sprintf("%s/v2/%s/manifests/%s", registryBase, imageName, tag)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// We expect a 401 with WWW-Authenticate header to tell us how to authenticate to the registry
	// if a token is required.
	if resp.StatusCode == http.StatusUnauthorized {
		wwwAuthHeader := resp.Header.Get("WWW-Authenticate")
		token, err := drc.getTokenUsingAuthHeader(wwwAuthHeader)
		if err != nil {
			return fmt.Errorf("failed to get token for %s: %w", imageName, err)
		}

		drc.token = token
		return nil
	}

	if resp.StatusCode == http.StatusOK {
		// A token is not needed if we can access the manifest.
		return nil
	}

	return fmt.Errorf("unexpected status code %d, %s", resp.StatusCode, url)
}

func (drc *DockerRegistryClient) ParseImageName(imageWithTag string) (string, string) {
	// Default to latest tag if no tag is provided
	tag := "latest"
	image := imageWithTag
	if strings.Contains(image, ":") {
		parts := strings.Split(image, ":")
		image = parts[0]
		tag = parts[1]
	}

	// Official images need the library prefix
	if !strings.Contains(image, "/") {
		image = "library/" + image
	}

	// Handle a registry URL being part of the image name (e.g. <url>/<org>/<name>)
	if parts := strings.SplitN(image, "/", 3); len(parts) >= 3 {
		image = fmt.Sprintf("%s/%s", parts[1], parts[2])
	}

	return image, tag
}

const defaultRegistryBase = "https://registry-1.docker.io"

func (drc *DockerRegistryClient) getRegistryBase() string {
	if drc.dockerImage.AuthConfig.ServerAddress != "" {
		return drc.dockerImage.AuthConfig.ServerAddress
	}
	return defaultRegistryBase
}

func (drc *DockerRegistryClient) getTokenUsingAuthHeader(wwwAuthHeader string) (string, error) {
	if wwwAuthHeader == "" {
		return "", fmt.Errorf("no WWW-Authenticate header")
	}

	imageName, _ := drc.ParseImageName(drc.dockerImage.Image)
	authURL, err := drc.parseWWWAuthenticate(wwwAuthHeader, imageName)
	if err != nil {
		return "", fmt.Errorf("failed to parse WWW-Authenticate header: %w", err)
	}

	if drc.dockerImage.AuthConfig.Username != "" && drc.dockerImage.AuthConfig.Password != "" {
		return drc.getTokenFromAuthEndpoint(authURL, true)
	}

	// No credentials provided, try anonymous access.
	return drc.getTokenFromAuthEndpoint(authURL, false)
}

var (
	realmRegex   = regexp.MustCompile(`realm="([^"]+)"`)
	serviceRegex = regexp.MustCompile(`service="([^"]+)"`)
	scopeRegex   = regexp.MustCompile(`scope="([^"]+)"`)
)

// Parse the WWW-Authenticate header and returns the auth endpoint.
func (drc *DockerRegistryClient) parseWWWAuthenticate(wwwAuth, image string) (string, error) {
	// The format of the WWW-Authenticate header is from the OAuth 2.0 spec.
	// Bearer realm="<url>",service="<service>",scope="<scope>"

	// The realm is required to get the auth endpoint.
	realmMatch := realmRegex.FindStringSubmatch(wwwAuth)
	if len(realmMatch) < 2 {
		return "", fmt.Errorf("could not parse realm from WWW-Authenticate: %s", wwwAuth)
	}

	authURL := realmMatch[1]

	// The service and scope are optional.
	serviceMatch := serviceRegex.FindStringSubmatch(wwwAuth)
	scopeMatch := scopeRegex.FindStringSubmatch(wwwAuth)

	params := make([]string, 0)
	if len(serviceMatch) >= 2 {
		params = append(params, fmt.Sprintf("service=%s", serviceMatch[1]))
	}
	if len(scopeMatch) >= 2 {
		params = append(params, fmt.Sprintf("scope=%s", scopeMatch[1]))
	} else {
		// Default scope
		params = append(params, fmt.Sprintf("scope=repository:%s:pull", image))
	}

	if len(params) > 0 {
		authURL += "?" + strings.Join(params, "&")
	}

	return authURL, nil
}

func (drc *DockerRegistryClient) getTokenFromAuthEndpoint(authURL string, withAuth bool) (string, error) {
	req, err := http.NewRequest("GET", authURL, nil)
	if err != nil {
		return "", err
	}

	// Add authentication if requested
	if withAuth && drc.dockerImage.AuthConfig.Username != "" && drc.dockerImage.AuthConfig.Password != "" {
		auth := fmt.Sprintf("%s:%s", drc.dockerImage.AuthConfig.Username, drc.dockerImage.AuthConfig.Password)
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		req.Header.Set("Authorization", "Basic "+encodedAuth)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: failed to get token from auth endpoint", resp.StatusCode)
	}

	var authResp RegistryAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", err
	}
	return authResp.Token, nil
}

func (drc *DockerRegistryClient) GetManifest(imageName, tag string) (*RegistryManifestV2, error) {
	registryBase := drc.getRegistryBase()
	url := fmt.Sprintf("%s/v2/%s/manifests/%s", registryBase, imageName, tag)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if drc.token != "" {
		req.Header.Set("Authorization", "Bearer "+drc.token)
	}
	req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json")
	req.Header.Add("Accept", "application/vnd.oci.image.index.v1+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v1+json")

	resp, err := drc.doRequestWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: failed to get manifest for %s", resp.StatusCode, imageName)
	}

	return drc.parseManifest(body, imageName, tag)
}

// doRequestWithRetry performs an HTTP request with automatic token refresh and retry logic.
// If a 401 is received, it attempts to refresh the token and retry once.
// If the retry also fails with 401, it returns an auth error.
func (drc *DockerRegistryClient) doRequestWithRetry(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	// If we get a 401, try to refresh the token and retry once
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		slog.Debug("unauthorized, attempting to refresh token")
		if err := drc.RefreshToken(); err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}

		// Update the request with the new token
		if drc.token != "" {
			req.Header.Set("Authorization", "Bearer "+drc.token)
		}

		// Retry the request
		resp2, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		// If we still get 401 after refresh, it's an auth issue
		if resp2.StatusCode == http.StatusUnauthorized {
			resp2.Body.Close()
			return nil, fmt.Errorf("insufficient permissions to access private repository - credentials required")
		}

		return resp2, nil
	}

	return resp, nil
}

func (drc *DockerRegistryClient) parseManifest(manifestData []byte, imageName, tag string) (*RegistryManifestV2, error) {
	var baseManifest struct {
		SchemaVersion int    `json:"schemaVersion"`
		MediaType     string `json:"mediaType"`
	}

	if err := json.Unmarshal(manifestData, &baseManifest); err != nil {
		return nil, fmt.Errorf("failed to parse base manifest: %w", err)
	}

	if baseManifest.SchemaVersion != 2 {
		return nil, fmt.Errorf("unsupported schema version: %d", baseManifest.SchemaVersion)
	}

	switch baseManifest.MediaType {
	case "application/vnd.oci.image.manifest.v1+json", "application/vnd.docker.distribution.manifest.v2+json":
		return drc.parseSingleManifest(manifestData)
	case "application/vnd.oci.image.index.v1+json", "application/vnd.docker.distribution.manifest.list.v2+json":
		return drc.parseMultipleManifest(manifestData, imageName, tag)
	default:
		return nil, fmt.Errorf("unknown manifest mediaType: %s", baseManifest.MediaType)
	}
}

func (drc *DockerRegistryClient) parseSingleManifest(manifestData []byte) (*RegistryManifestV2, error) {
	var manifest RegistryManifestV2
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse single manifest: %w", err)
	}
	return &manifest, nil
}

// parseMultipleManifest finds the manifest for the current architecture
func (drc *DockerRegistryClient) parseMultipleManifest(manifestData []byte, imageName, tag string) (*RegistryManifestV2, error) {
	var manifestList RegistryManifestList
	if err := json.Unmarshal(manifestData, &manifestList); err != nil {
		return nil, fmt.Errorf("failed to parse manifest list: %w", err)
	}

	for _, manifest := range manifestList.Manifests {
		if manifest.Platform.Architecture != runtime.GOARCH {
			continue
		}

		return drc.GetManifest(imageName, manifest.Digest)
	}

	return nil, fmt.Errorf("no manifest found for architecture: %s", runtime.GOARCH)
}

func (drc *DockerRegistryClient) GetLayerData(image, digest string) ([]byte, error) {
	registryBase := drc.getRegistryBase()
	url := fmt.Sprintf("%s/v2/%s/blobs/%s", registryBase, image, digest)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if drc.token != "" {
		req.Header.Set("Authorization", "Bearer "+drc.token)
	}

	resp, err := drc.doRequestWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: failed to fetch blob", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// Registry types for Docker manifest and layer data

type RegistryAuthResponse struct {
	Token string `json:"token"`
}

type RegistryLayer struct {
	MediaType string `json:"mediaType"`
	Digest    string `json:"digest"`
}

// createIdFromParent creates a unique, deterministic ID for the layer based on its parent ID and digest
func (l *RegistryLayer) CreateIdFromParent(parentId string) string {
	h := sha256.New()
	h.Write([]byte(parentId + "\n" + l.Digest))
	return fmt.Sprintf("%x", h.Sum(nil))
}

type RegistryManifestV2 struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		Digest string `json:"digest"`
	} `json:"config"`
	Layers []RegistryLayer `json:"layers"`
}

type RegistryManifestList struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Manifests     []struct {
		Digest   string `json:"digest"`
		Platform struct {
			Architecture string `json:"architecture"`
		} `json:"platform"`
	} `json:"manifests"`
}

type RegistryManifestEntry struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
}
