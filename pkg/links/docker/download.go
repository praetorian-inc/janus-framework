package docker

import (
	"archive/tar"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/types"
)

// DockerDownloadLink downloads Docker images from a registry without requiring the Docker daemon
// and saves them as tar files.
type DockerDownloadLink struct {
	*chain.Base
	outDir      string
	token       string
	dockerImage *types.DockerImage
}

func NewDockerDownload(configs ...cfg.Config) chain.Link {
	dd := &DockerDownloadLink{}
	dd.Base = chain.NewBase(dd, configs...)
	return dd
}

func (dd *DockerDownloadLink) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("output", "output directory").
			WithShortcode("o").
			WithDefault("docker-images"),
	}
}

func (dd *DockerDownloadLink) Initialize() error {
	dir, err := cfg.As[string](dd.Arg("output"))
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	dd.outDir = dir
	return nil
}

func (dd *DockerDownloadLink) Process(dockerImage *types.DockerImage) error {
	if dockerImage.Image == "" {
		return fmt.Errorf("Docker image name is required")
	}
	dd.dockerImage = dockerImage

	outFile, err := createOutputFile(dd.outDir, dockerImage.Image)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	if err := dd.downloadImage(dockerImage, outFile); err != nil {
		return fmt.Errorf("failed to download image %s: %w", dockerImage.Image, err)
	}

	dockerImage.LocalPath = outFile.Name()
	return dd.Send(dockerImage)
}

// This is an adapted version of https://github.com/moby/moby/blob/master/contrib/download-frozen-image-v2.sh
func (dd *DockerDownloadLink) downloadImage(dockerImage *types.DockerImage, outputTarFile *os.File) error {
	imageName, tag := dd.parseImageName(dockerImage.Image)

	// Image components will be downloaded here before being added to a tar file
	tempDir, err := os.MkdirTemp("", "docker-download-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	if err := dd.refreshToken(dockerImage); err != nil {
		return err
	}

	manifestData, err := dd.getManifest(imageName, tag)
	if err != nil {
		return err
	}

	if err := dd.downloadImageFromManifest(manifestData, imageName, tag, tempDir); err != nil {
		return err
	}

	return dd.createTarFile(tempDir, outputTarFile)
}

func (dd *DockerDownloadLink) parseImageName(imageWithTag string) (string, string) {
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

func (dd *DockerDownloadLink) refreshToken(dockerImage *types.DockerImage) error {
	imageName, tag := dd.parseImageName(dockerImage.Image)

	// Reset the token so we know if a refresh is needed versus auth failure
	dd.token = ""

	// Always try to access the registry first to get the WWW-Authenticate header or see if a token is required.
	registryBase := dd.getRegistryBase(dockerImage)
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
		token, err := dd.getTokenUsingAuthHeader(dockerImage, wwwAuthHeader)
		if err != nil {
			return fmt.Errorf("failed to get token for %s: %w", imageName, err)
		}

		dd.token = token
		return nil
	}

	if resp.StatusCode == http.StatusOK {
		// A token is not needed if we can access the manifest.
		return nil
	}

	return fmt.Errorf("unexpected status code %d, %s", resp.StatusCode, url)
}

const defaultRegistryBase = "https://registry-1.docker.io"

func (dd *DockerDownloadLink) getRegistryBase(dockerImage *types.DockerImage) string {
	if dockerImage.AuthConfig.ServerAddress != "" {
		return dockerImage.AuthConfig.ServerAddress
	}
	return defaultRegistryBase
}

func (dd *DockerDownloadLink) getTokenUsingAuthHeader(dockerImage *types.DockerImage, wwwAuthHeader string) (string, error) {
	if wwwAuthHeader == "" {
		return "", fmt.Errorf("no WWW-Authenticate header")
	}

	imageName, _ := dd.parseImageName(dockerImage.Image)
	authURL, err := dd.parseWWWAuthenticate(wwwAuthHeader, imageName)
	if err != nil {
		return "", fmt.Errorf("failed to parse WWW-Authenticate header: %w", err)
	}

	if dockerImage.AuthConfig.Username != "" && dockerImage.AuthConfig.Password != "" {
		return dd.getTokenFromAuthEndpoint(authURL, dockerImage, true)
	}

	// No credentials provided, try anonymous access.
	return dd.getTokenFromAuthEndpoint(authURL, dockerImage, false)
}

var (
	realmRegex   = regexp.MustCompile(`realm="([^"]+)"`)
	serviceRegex = regexp.MustCompile(`service="([^"]+)"`)
	scopeRegex   = regexp.MustCompile(`scope="([^"]+)"`)
)

// Parse the WWW-Authenticate header and returns the auth endpoint.
func (dd *DockerDownloadLink) parseWWWAuthenticate(wwwAuth, image string) (string, error) {
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

func (dd *DockerDownloadLink) getTokenFromAuthEndpoint(authURL string, dockerImage *types.DockerImage, withAuth bool) (string, error) {
	req, err := http.NewRequest("GET", authURL, nil)
	if err != nil {
		return "", err
	}

	// Add authentication if requested
	if withAuth && dockerImage.AuthConfig.Username != "" && dockerImage.AuthConfig.Password != "" {
		auth := fmt.Sprintf("%s:%s", dockerImage.AuthConfig.Username, dockerImage.AuthConfig.Password)
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

func (dd *DockerDownloadLink) getManifest(image, digest string) ([]byte, error) {
	registryBase := dd.getRegistryBase(dd.dockerImage)
	url := fmt.Sprintf("%s/v2/%s/manifests/%s", registryBase, image, digest)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if dd.token != "" {
		req.Header.Set("Authorization", "Bearer "+dd.token)
	}
	req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json")
	req.Header.Add("Accept", "application/vnd.oci.image.index.v1+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v1+json")

	resp, err := dd.doRequestWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: failed to get manifest for %s", resp.StatusCode, image)
	}

	return body, nil
}

// doRequestWithRetry performs an HTTP request with automatic token refresh and retry logic.
// If a 401 is received, it attempts to refresh the token and retry once.
// If the retry also fails with 401, it returns an auth error.
func (dd *DockerDownloadLink) doRequestWithRetry(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	// If we get a 401, try to refresh the token and retry once
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		slog.Debug("unauthorized, attempting to refresh token")
		if err := dd.refreshToken(dd.dockerImage); err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}

		// Update the request with the new token
		if dd.token != "" {
			req.Header.Set("Authorization", "Bearer "+dd.token)
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

func (dd *DockerDownloadLink) downloadImageFromManifest(manifestData []byte, imageName, tag, outputDir string) error {
	var baseManifest struct {
		SchemaVersion int    `json:"schemaVersion"`
		MediaType     string `json:"mediaType"`
	}

	if err := json.Unmarshal(manifestData, &baseManifest); err != nil {
		return err
	}

	if baseManifest.SchemaVersion != 2 {
		return fmt.Errorf("unsupported schema version: %d", baseManifest.SchemaVersion)
	}

	var manifestEntry RegistryManifestEntry
	var err error

	switch baseManifest.MediaType {
	case "application/vnd.oci.image.manifest.v1+json", "application/vnd.docker.distribution.manifest.v2+json":
		manifestEntry, err = dd.handleSingleManifestV2(manifestData, imageName, tag, outputDir)
		if err != nil {
			return err
		}

	case "application/vnd.oci.image.index.v1+json", "application/vnd.docker.distribution.manifest.list.v2+json":
		manifestEntry, err = dd.handleMultipleManifestV2(manifestData, imageName, tag, outputDir)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown manifest mediaType: %s", baseManifest.MediaType)
	}

	manifestFile := filepath.Join(outputDir, "manifest.json")
	file, err := os.Create(manifestFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode([]RegistryManifestEntry{manifestEntry})
}

func (dd *DockerDownloadLink) handleSingleManifestV2(manifestJson []byte, image, tag, outputDir string) (RegistryManifestEntry, error) {
	var manifest RegistryManifestV2
	if err := json.Unmarshal(manifestJson, &manifest); err != nil {
		return RegistryManifestEntry{}, err
	}

	// Download config
	configDigest := manifest.Config.Digest
	imageId := strings.TrimPrefix(configDigest, "sha256:")
	configFile := imageId + ".json"

	if err := dd.fetchBlob(image, configDigest, filepath.Join(outputDir, configFile)); err != nil {
		return RegistryManifestEntry{}, err
	}

	var layerFiles []string
	var layerId string

	for _, layer := range manifest.Layers {
		newLayerId, layerTar, err := dd.downloadLayer(layer, layerId, image, outputDir)
		if err != nil {
			return RegistryManifestEntry{}, fmt.Errorf("failed to download layer: %w", err)
		}

		layerId = newLayerId
		layerFiles = append(layerFiles, layerTar)
	}

	// Create manifest entry
	repoTag := strings.TrimPrefix(image, "library/") + ":" + tag
	manifestEntry := RegistryManifestEntry{
		Config:   configFile,
		RepoTags: []string{repoTag},
		Layers:   layerFiles,
	}

	return manifestEntry, nil
}

func (dd *DockerDownloadLink) downloadLayer(layer RegistryLayer, parentId, image, outputDir string) (string, string, error) {
	// Create a fake layer ID
	layerId := layer.createIdFromParent(parentId)

	// Download layer data
	if layer.MediaType != "application/vnd.oci.image.layer.v1.tar+gzip" && layer.MediaType != "application/vnd.docker.image.rootfs.diff.tar.gzip" {
		return layerId, "", fmt.Errorf("unknown layer mediaType: %s", layer.MediaType)
	}

	layerTar := fmt.Sprintf("%s.tar", layerId)
	layerPath := filepath.Join(outputDir, layerTar)
	if _, err := os.Stat(layerPath); err == nil {
		return layerId, layerTar, nil
	}

	return layerId, layerTar, dd.fetchBlob(image, layer.Digest, layerPath)
}

func (dd *DockerDownloadLink) handleMultipleManifestV2(manifestData []byte, imageName, tag, outputDir string) (RegistryManifestEntry, error) {
	var manifestList RegistryManifestList
	if err := json.Unmarshal(manifestData, &manifestList); err != nil {
		return RegistryManifestEntry{}, err
	}

	var manifestEntry RegistryManifestEntry

	// Search through all manifests for the one matching our architecture
	for _, manifest := range manifestList.Manifests {
		if manifest.Platform.Architecture != runtime.GOARCH {
			continue
		}

		subManifestData, err := dd.getManifest(imageName, manifest.Digest)
		if err != nil {
			return RegistryManifestEntry{}, err
		}

		manifestEntry, err = dd.handleSingleManifestV2(subManifestData, imageName, tag, outputDir)
		if err != nil {
			return RegistryManifestEntry{}, err
		}

		break
	}

	if manifestEntry.Config == "" {
		return RegistryManifestEntry{}, fmt.Errorf("manifest for %s not found", runtime.GOARCH)
	}

	return manifestEntry, nil
}

func (dd *DockerDownloadLink) fetchBlob(image, digest, targetFile string) error {
	registryBase := dd.getRegistryBase(dd.dockerImage)
	url := fmt.Sprintf("%s/v2/%s/blobs/%s", registryBase, image, digest)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	if dd.token != "" {
		req.Header.Set("Authorization", "Bearer "+dd.token)
	}

	resp, err := dd.doRequestWithRetry(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: failed to fetch blob", resp.StatusCode)
	}

	file, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

func (dd *DockerDownloadLink) createTarFile(sourceDir string, tarFile *os.File) error {
	tarWriter := tar.NewWriter(tarFile)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		header, err := tar.FileInfoHeader(info, relPath)
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer file.Close()

		if _, err := io.Copy(tarWriter, file); err != nil {
			return fmt.Errorf("failed to copy file content to tar: %w", err)
		}

		return nil
	})
}
