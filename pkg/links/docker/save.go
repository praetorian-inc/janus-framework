package docker

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/client"
	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	dockerTypes "github.com/praetorian-inc/janus-framework/pkg/types/docker"
)

type DockerSave struct {
	*chain.Base
	outDir string
}

func NewDockerSave(configs ...cfg.Config) chain.Link {
	dsl := &DockerSave{}
	dsl.Base = chain.NewBase(dsl, configs...)
	return dsl
}

func (dsl *DockerSave) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("output", "output directory to save images to"),
	}
}

func (dsl *DockerSave) Initialize() error {
	dir, err := cfg.As[string](dsl.Arg("output"))
	if err != nil {
		return err
	}

	if dir == "" {
		dir = filepath.Join(os.TempDir(), ".janus-docker-images")
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	dsl.outDir = dir

	return nil
}

func (dsl *DockerSave) Process(imageContext dockerTypes.DockerImage) error {
	isPublicImage := strings.Contains(imageContext.AuthConfig.ServerAddress, "public.ecr.aws")

	var dockerClient *client.Client
	var err error

	if !isPublicImage {
		dockerClient, err = NewAuthenticatedClient(dsl.Context(), imageContext, client.FromEnv)
	} else {
		dockerClient, err = NewUnauthenticatedClient(dsl.Context(), client.FromEnv)
	}

	if err != nil {
		return err
	}

	defer dockerClient.Close()

	imageID := imageContext.Image

	defer removeImage(dsl.Context(), dockerClient, imageID)

	outFile, err := createOutputFile(dsl.outDir, imageID)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	reader, err := dockerClient.ImageSave(dsl.Context(), []string{imageID})
	if err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}
	defer reader.Close()

	if _, err := io.Copy(outFile, reader); err != nil {
		return fmt.Errorf("failed to copy image to output file: %w", err)
	}

	imageContext.LocalPath = outFile.Name()

	return dsl.Send(&imageContext)
}
