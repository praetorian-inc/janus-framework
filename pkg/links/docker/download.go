package docker

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/types"
)

const (
	registryBase = "https://registry-1.docker.io"
	authBase     = "https://auth.docker.io"
	authService  = "registry.docker.io"
)

type AuthResponse struct {
	Token string `json:"token"`
}

type ManifestV2 struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		Digest string `json:"digest"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
	} `json:"layers"`
}

type ManifestList struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Manifests     []struct {
		Digest   string `json:"digest"`
		Platform struct {
			Architecture string `json:"architecture"`
		} `json:"platform"`
	} `json:"manifests"`
}

type ManifestEntry struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
}

// DockerDownloadLink downloads Docker images from a registry without requiring the Docker daemon 
// and saves them as tar files.
type DockerDownloadLink struct {
	*chain.Base
	outputTarFile string
}

func NewDockerDownload(configs ...cfg.Config) chain.Link {
	dd := &DockerDownloadLink{}
	dd.Base = chain.NewBase(dd, configs...)
	return dd
}

func (dd *DockerDownloadLink) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("output", "Full path to the tar file to save the downloaded image").WithDefault(""),
	}
}

func (dd *DockerDownloadLink) Initialize() error {
	var err error
	dd.outputTarFile, err = cfg.As[string](dd.Arg("output"))
	if err != nil {
		return fmt.Errorf("failed to get output tar file path: %w", err)
	}

	if dd.outputTarFile == "" {
		tempDir, err := os.MkdirTemp("", "docker-download-*")
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
		dd.outputTarFile = filepath.Join(tempDir, "image.tar")
	}

	return nil
}

func (dd *DockerDownloadLink) Process(dockerImage *types.DockerImage) error {
	if dockerImage.Image == "" {
		return fmt.Errorf("Docker image name is required")
	}

	slog.Info("downloading Docker image", "image", dockerImage.Image, "output", dd.outputTarFile)
	if err := dd.downloadImage(dockerImage.Image, dd.outputTarFile); err != nil {
		return fmt.Errorf("failed to download image %s: %w", dockerImage.Image, err)
	}

	dockerImage.LocalPath = dd.outputTarFile
	return dd.Send(dockerImage)
}

// This is an adapted version of https://github.com/moby/moby/blob/master/contrib/download-frozen-image-v2.sh
func (dd *DockerDownloadLink) downloadImage(image, outputTarFile string) error {
	// Default to latest tag if no tag is provided
	tag := "latest"
	if strings.Contains(image, ":") {
		parts := strings.Split(image, ":")
		image = parts[0]
		tag = parts[1]
	}

	// Official images need the library prefix
	if !strings.Contains(image, "/") {
		image = "library/" + image
	}

	// Image components will be downloaded here before being added to a tar file
	tempDir, err := os.MkdirTemp("", "docker-download-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	token, err := dd.getToken(image)
	if err != nil {
		return err
	}

	manifestData, err := dd.getManifest(token, image, tag)
	if err != nil {
		return err
	}

	var baseManifest struct {
		SchemaVersion int    `json:"schemaVersion"`
		MediaType     string `json:"mediaType"`
	}

	if err := json.Unmarshal(manifestData, &baseManifest); err != nil {
		return err
	}

	var manifestEntry ManifestEntry

	switch baseManifest.SchemaVersion {
	case 2:
		switch baseManifest.MediaType {
		case "application/vnd.oci.image.manifest.v1+json", "application/vnd.docker.distribution.manifest.v2+json":
			manifestEntry, err = dd.handleSingleManifestV2(manifestData, image, tag, tempDir)
			if err != nil {
				return err
			}

		case "application/vnd.oci.image.index.v1+json", "application/vnd.docker.distribution.manifest.list.v2+json":
			var manifestList ManifestList
			if err := json.Unmarshal(manifestData, &manifestList); err != nil {
				return err
			}

			for _, manifest := range manifestList.Manifests {
				if manifest.Platform.Architecture == runtime.GOARCH {
					subManifestData, err := dd.getManifest(token, image, manifest.Digest)
					if err != nil {
						return err
					}

					manifestEntry, err = dd.handleSingleManifestV2(subManifestData, image, tag, tempDir)
					if err != nil {
						return err
					}
					break
				}
			}

			if manifestEntry.Config == "" {
				return fmt.Errorf("manifest for %s not found", runtime.GOARCH)
			}

		default:
			return fmt.Errorf("unknown manifest mediaType: %s", baseManifest.MediaType)
		}

	default:
		return fmt.Errorf("unsupported schema version: %d", baseManifest.SchemaVersion)
	}

	manifestFile := filepath.Join(tempDir, "manifest.json")
	file, err := os.Create(manifestFile)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode([]ManifestEntry{manifestEntry}); err != nil {
		return err
	}

	return dd.createTarFile(tempDir, outputTarFile)
}

func (dd *DockerDownloadLink) getToken(image string) (string, error) {
	url := fmt.Sprintf("%s/token?service=%s&scope=repository:%s:pull", authBase, authService, image)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", err
	}

	return authResp.Token, nil
}

func (dd *DockerDownloadLink) fetchBlob(token, image, digest, targetFile string) error {
	url := fmt.Sprintf("%s/v2/%s/blobs/%s", registryBase, image, digest)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Handle redirects (status 3xx)
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := resp.Header.Get("Location")
		if location == "" {
			return fmt.Errorf("redirect without location header")
		}

		resp2, err := http.Get(location)
		if err != nil {
			return err
		}
		defer resp2.Body.Close()
		resp = resp2
	}

	file, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

func (dd *DockerDownloadLink) getManifest(token, image, digest string) ([]byte, error) {
	url := fmt.Sprintf("%s/v2/%s/manifests/%s", registryBase, image, digest)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json")
	req.Header.Add("Accept", "application/vnd.oci.image.index.v1+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v1+json")

	resp, err := http.DefaultClient.Do(req)
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

func (dd *DockerDownloadLink) handleSingleManifestV2(manifestJson []byte, image, tag, outputDir string) (ManifestEntry, error) {
	var manifest ManifestV2
	if err := json.Unmarshal(manifestJson, &manifest); err != nil {
		return ManifestEntry{}, err
	}

	token, err := dd.getToken(image)
	if err != nil {
		return ManifestEntry{}, err
	}

	// Download config
	configDigest := manifest.Config.Digest
	imageId := strings.TrimPrefix(configDigest, "sha256:")
	configFile := imageId + ".json"

	if err := dd.fetchBlob(token, image, configDigest, filepath.Join(outputDir, configFile)); err != nil {
		return ManifestEntry{}, err
	}

	var layerFiles []string
	var layerId string

	for _, layer := range manifest.Layers {
		parentId := layerId

		// Create fake layer ID
		h := sha256.New()
		h.Write([]byte(parentId + "\n" + layer.Digest))
		layerId = fmt.Sprintf("%x", h.Sum(nil))

		layerDir := filepath.Join(outputDir, layerId)
		if err := os.MkdirAll(layerDir, 0755); err != nil {
			return ManifestEntry{}, err
		}

		// Download layer data
		switch layer.MediaType {
		case "application/vnd.oci.image.layer.v1.tar+gzip", "application/vnd.docker.image.rootfs.diff.tar.gzip":
			layerTar := filepath.Join(layerId, "layer.tar")
			layerFiles = append(layerFiles, layerTar)

			layerPath := filepath.Join(outputDir, layerTar)
			if _, err := os.Stat(layerPath); err == nil {
				continue
			}

			if err := dd.fetchBlob(token, image, layer.Digest, layerPath); err != nil {
				return ManifestEntry{}, err
			}

		default:
			return ManifestEntry{}, fmt.Errorf("unknown layer mediaType: %s", layer.MediaType)
		}
	}

	// Create manifest entry
	repoTag := strings.TrimPrefix(image, "library/") + ":" + tag
	manifestEntry := ManifestEntry{
		Config:   configFile,
		RepoTags: []string{repoTag},
		Layers:   layerFiles,
	}

	return manifestEntry, nil
}

func (dd *DockerDownloadLink) createTarFile(sourceDir, outputTarFile string) error {
	tarFile, err := os.Create(outputTarFile)
	if err != nil {
		return fmt.Errorf("failed to create tar file: %w", err)
	}
	defer tarFile.Close()

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

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", path, err)
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return fmt.Errorf("failed to copy file content to tar: %w", err)
			}
		}

		return nil
	})
}
