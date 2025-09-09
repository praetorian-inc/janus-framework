package docker

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/docker/docker/api/types/registry"
	"github.com/praetorian-inc/janus-framework/pkg/types"
)

type DockerManifest struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
}

type DockerImage struct {
	AuthConfig registry.AuthConfig
	Image      string
	LocalPath  string
	Manifest   *RegistryManifestV2
}

type DockerLayer struct {
	*DockerImage
	Digest string
	Data   []byte
}

func (i *DockerImage) ToNPInputs() ([]types.NPInput, error) {
	if i.LocalPath == "" {
		return nil, fmt.Errorf("local path required to convert to []types.NPInput")
	}

	imageFile, err := os.Open(i.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file: %w", err)
	}
	defer imageFile.Close()

	manifest, err := i.extractManifest(imageFile)
	if err != nil {
		return nil, fmt.Errorf("failed to extract manifest: %w", err)
	}

	imageFile.Seek(0, io.SeekStart)
	tarReader := tar.NewReader(imageFile)

	return i.createNPInputs(tarReader, manifest)
}

func (i *DockerImage) extractManifest(imageFile *os.File) ([]DockerManifest, error) {
	tarReader := tar.NewReader(imageFile)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("failed to read tar header: %w", err)
		}

		if header.Name == "manifest.json" {
			return i.parseManifest(tarReader)
		}
	}

	return nil, nil
}

func (i *DockerImage) parseManifest(reader io.Reader) ([]DockerManifest, error) {
	manifestBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed reading manifest: %w", err)
	}

	var manifest []DockerManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("failed parsing manifest: %w", err)
	}

	if len(manifest) == 0 {
		return nil, fmt.Errorf("no manifest found in image")
	}

	return manifest, nil
}

func (i *DockerImage) createNPInputs(tarReader *tar.Reader, manifest []DockerManifest) ([]types.NPInput, error) {
	totalInputs := []types.NPInput{}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return totalInputs, nil
		}

		if err != nil {
			return nil, fmt.Errorf("failed reading tar: %w", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		fileName, isLayer := i.extractFileName(header, manifest)

		var inputs []types.NPInput
		if isLayer {
			inputs, err = i.processLayer(tarReader, fileName)
		} else {
			inputs, err = i.processFile(tarReader, fileName)
		}

		if err != nil {
			slog.Debug(fmt.Sprintf("failed processing layer: %v", err))
		}

		totalInputs = append(totalInputs, inputs...)
	}
}

func (i *DockerImage) extractFileName(header *tar.Header, manifest []DockerManifest) (string, bool) {
	isLayer := false
	layerName := ""
	for _, m := range manifest {
		for _, layer := range m.Layers {
			if header.Name == layer {
				isLayer = true
				layerName = layer
				break
			}
		}

		if header.Name == m.Config {
			isLayer = false
			layerName = m.Config
			break
		}

		if isLayer {
			break
		}
	}

	return layerName, isLayer
}

func (i *DockerImage) processLayer(tarReader *tar.Reader, layerName string) ([]types.NPInput, error) {
	var npInputs []types.NPInput

	err := i.ProcessLayerWithCallback(tarReader, layerName, func(npInput *types.NPInput) error {
		npInputs = append(npInputs, *npInput)
		return nil
	})

	return npInputs, err
}

type LayerProcessCallback func(npInput *types.NPInput) error

// ProcessLayerWithCallback processes a layer using a callback for each NPInput.
// This eliminates duplication between streaming and collecting layer processing.
func (i *DockerImage) ProcessLayerWithCallback(reader io.Reader, layerName string, callback LayerProcessCallback) error {
	layerReader, cleanup, err := i.getLayerReader(reader, layerName)
	if err != nil {
		return err
	}
	if layerReader == nil {
		return nil // Empty layer
	}
	defer cleanup()

	for {
		header, err := layerReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Debug("failed reading layer header", "error", err)
			continue
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		content, err := io.ReadAll(layerReader)
		if err != nil {
			slog.Debug("failed reading file", "error", err)
			continue
		}

		if len(content) == 0 {
			continue
		}

		npInput := &types.NPInput{
			ContentBase64: base64.StdEncoding.EncodeToString(content),
			Provenance: types.NPProvenance{
				Platform:     "docker",
				ResourceType: "layer",
				ResourceID:   i.Image,
				Region:       fmt.Sprintf("%s,%s", layerName, header.Name),
			},
		}

		if err := callback(npInput); err != nil {
			return err
		}
	}

	return nil
}

// getLayerReader creates the appropriate reader for a Docker layer, handling both gzipped and uncompressed layers.
// It returns a cleanup function that should be called to close the reader.
func (i *DockerImage) getLayerReader(reader io.Reader, layerName string) (*tar.Reader, func() error, error) {
	var cleanup func() error = func() error { return nil }

	magicBytes, err := extractMagicBytes(reader, 2)
	if err != nil {
		return nil, cleanup, err
	}
	if len(magicBytes) == 0 {
		slog.Debug("empty layer", "layer", layerName)
		return nil, cleanup, nil
	}

	// Create a reader that starts with the header bytes we already read,
	// followed by the rest of the tar entry
	headerReader := bytes.NewReader(magicBytes)
	combinedReader := io.MultiReader(headerReader, reader)

	var layerReader io.Reader

	if isGzip(magicBytes) {
		gzipReader, err := gzip.NewReader(combinedReader)
		if err != nil {
			return nil, cleanup, fmt.Errorf("failed to create gzip reader for layer %s: %w", layerName, err)
		}
		layerReader = gzipReader
		cleanup = gzipReader.Close
	} else {
		layerReader = combinedReader
	}

	return tar.NewReader(layerReader), cleanup, nil
}

func extractMagicBytes(reader io.Reader, numBytes int) ([]byte, error) {
	magicBytes := make([]byte, numBytes)
	n, err := reader.Read(magicBytes)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to read header: %w", err)
	}

	return magicBytes[:n], nil
}

var gzipMagicBytes = []byte{0x1f, 0x8b}

func isGzip(magicBytes []byte) bool {
	return len(magicBytes) >= 2 && bytes.Equal(magicBytes[:2], gzipMagicBytes)
}

func (i *DockerImage) processFile(tarReader *tar.Reader, fileName string) ([]types.NPInput, error) {
	content, err := io.ReadAll(tarReader)

	if err != nil {
		return nil, fmt.Errorf("failed reading file %s: %w", fileName, err)
	}

	if len(content) == 0 {
		return nil, fmt.Errorf("file %q is empty", fileName)
	}

	input := types.NPInput{
		ContentBase64: base64.StdEncoding.EncodeToString(content),
		Provenance: types.NPProvenance{
			Platform:     "docker",
			ResourceType: "image",
			ResourceID:   i.Image,
			Region:       fmt.Sprintf("file:%s", fileName),
		},
	}

	return []types.NPInput{input}, nil
}
