package docker

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	dockerTypes "github.com/praetorian-inc/janus-framework/pkg/types/docker"
)

// DockerDownloadLink downloads Docker images from a registry without requiring the Docker daemon
// and saves them as tar files.
type DockerDownloadLink struct {
	*chain.Base
	outDir         string
	registryClient dockerTypes.DockerRegistryClient
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

func (dd *DockerDownloadLink) Process(dockerImage *dockerTypes.DockerImage) error {
	if dockerImage.Image == "" {
		return fmt.Errorf("Docker image name is required")
	}
	dd.registryClient = *dockerTypes.NewDockerRegistryClient(dockerImage)

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
func (dd *DockerDownloadLink) downloadImage(dockerImage *dockerTypes.DockerImage, outputTarFile *os.File) error {
	imageName, tag := dd.registryClient.ParseImageName(dockerImage.Image)

	// Image components will be downloaded here before being added to a tar file
	tempDir, err := os.MkdirTemp("", "docker-download-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	if err := dd.registryClient.RefreshToken(); err != nil {
		return err
	}

	manifest, err := dd.registryClient.GetManifest(imageName, tag)
	if err != nil {
		return err
	}

	if err := dd.downloadImageFromManifest(manifest, imageName, tag, tempDir); err != nil {
		return err
	}

	return dd.createTarFile(tempDir, outputTarFile)
}

func (dd *DockerDownloadLink) downloadImageFromManifest(manifest *dockerTypes.RegistryManifestV2, imageName, tag, outputDir string) error {
	// Download config
	configDigest := manifest.Config.Digest
	imageId := strings.TrimPrefix(configDigest, "sha256:")
	configFile := imageId + ".json"

	configData, err := dd.registryClient.GetLayerData(imageName, configDigest)
	if err != nil {
		return err
	}

	configPath := filepath.Join(outputDir, configFile)
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return err
	}

	// Download layers
	var layerFiles []string

	for _, layer := range manifest.Layers {
		layerTar, err := dd.downloadLayer(layer, imageName, outputDir)
		if err != nil {
			return fmt.Errorf("failed to download layer: %w", err)
		}

		layerFiles = append(layerFiles, layerTar)
	}

	// Create manifest entry
	repoTag := strings.TrimPrefix(imageName, "library/") + ":" + tag
	manifestEntry := dockerTypes.DockerManifest{
		Config:   configFile,
		RepoTags: []string{repoTag},
		Layers:   layerFiles,
	}

	// Write manifest.json
	manifestFile := filepath.Join(outputDir, "manifest.json")
	file, err := os.Create(manifestFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode([]dockerTypes.DockerManifest{manifestEntry})
}

func (dd *DockerDownloadLink) downloadLayer(layer dockerTypes.RegistryLayer, image, outputDir string) (string, error) {
	if layer.MediaType != "application/vnd.oci.image.layer.v1.tar+gzip" && layer.MediaType != "application/vnd.docker.image.rootfs.diff.tar.gzip" {
		return "", fmt.Errorf("unknown layer mediaType: %s", layer.MediaType)
	}

	layerId := strings.TrimPrefix(layer.Digest, "sha256:")
	layerTar := fmt.Sprintf("%s.tar", layerId)
	layerPath := filepath.Join(outputDir, layerTar)

	if _, err := os.Stat(layerPath); err == nil {
		return layerTar, nil
	}

	layerData, err := dd.registryClient.GetLayerData(image, layer.Digest)
	if err != nil {
		return layerTar, err
	}

	return layerTar, os.WriteFile(layerPath, layerData, 0644)
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

// DockerGetLayersLink downloads manifest and emits individual layers of a Docker image
type DockerGetLayersLink struct {
	*chain.Base
	registryClient dockerTypes.DockerRegistryClient
}

func NewDockerGetLayers(configs ...cfg.Config) chain.Link {
	dgl := &DockerGetLayersLink{}
	dgl.Base = chain.NewBase(dgl, configs...)
	return dgl
}

func (dgl *DockerGetLayersLink) Process(dockerImage *dockerTypes.DockerImage) error {
	if dockerImage.Image == "" {
		return fmt.Errorf("Docker image name is required")
	}

	dgl.registryClient = *dockerTypes.NewDockerRegistryClient(dockerImage)
	imageName, tag := dgl.registryClient.ParseImageName(dockerImage.Image)

	if err := dgl.registryClient.RefreshToken(); err != nil {
		return err
	}

	// Download and parse manifest
	manifest, err := dgl.registryClient.GetManifest(imageName, tag)
	if err != nil {
		return fmt.Errorf("failed to download manifest for %s: %w", dockerImage.Image, err)
	}

	dockerImage.Manifest = manifest

	if err := dgl.emitConfig(dockerImage, manifest); err != nil {
		return err
	}

	return dgl.emitLayers(dockerImage, manifest)
}

func (dgl *DockerGetLayersLink) emitConfig(dockerImage *dockerTypes.DockerImage, manifest *dockerTypes.RegistryManifestV2) error {
	if manifest.Config.Digest != "" {
		configInput := &dockerTypes.DockerLayer{
			DockerImage: dockerImage,
			Digest:      manifest.Config.Digest,
		}
		if err := dgl.Send(configInput); err != nil {
			return fmt.Errorf("failed to send config input: %w", err)
		}
	}
	return nil
}

func (dgl *DockerGetLayersLink) emitLayers(dockerImage *dockerTypes.DockerImage, manifest *dockerTypes.RegistryManifestV2) error {
	for _, layer := range manifest.Layers {
		layerInput := &dockerTypes.DockerLayer{
			DockerImage: dockerImage,
			Digest:      layer.Digest,
		}
		if err := dgl.Send(layerInput); err != nil {
			return fmt.Errorf("failed to send layer input: %w", err)
		}
	}
	return nil
}

// DockerDownloadLayerLink downloads a single layer from a Docker registry
type DockerDownloadLayerLink struct {
	*chain.Base
	registryClient dockerTypes.DockerRegistryClient
}

func NewDockerDownloadLayer(configs ...cfg.Config) chain.Link {
	ddl := &DockerDownloadLayerLink{}
	ddl.Base = chain.NewBase(ddl, configs...)
	return ddl
}

func (ddl *DockerDownloadLayerLink) Process(layer *dockerTypes.DockerLayer) error {
	if layer.DockerImage == nil || layer.Digest == "" {
		return fmt.Errorf("DockerImage and Digest are required")
	}

	ddl.registryClient = *dockerTypes.NewDockerRegistryClient(layer.DockerImage)
	imageName, _ := ddl.registryClient.ParseImageName(layer.DockerImage.Image)

	if err := ddl.registryClient.RefreshToken(); err != nil {
		return err
	}

	layerData, err := ddl.registryClient.GetLayerData(imageName, layer.Digest)
	if err != nil {
		return fmt.Errorf("failed to download layer %s: %w", layer.Digest, err)
	}

	layer.Data = layerData
	return ddl.Send(layer)
}
