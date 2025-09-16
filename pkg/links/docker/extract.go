package docker

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/types"
	dockerTypes "github.com/praetorian-inc/janus-framework/pkg/types/docker"
)

// DockerLayerToNP converts downloaded layer data to NoseyParker inputs
type DockerLayerToNP struct {
	*chain.Base
}

func NewDockerLayerToNP(configs ...cfg.Config) chain.Link {
	dlnp := &DockerLayerToNP{}
	dlnp.Base = chain.NewBase(dlnp, configs...)
	return dlnp
}

func (dlnp *DockerLayerToNP) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[int]("max-file-size", "maximum file size to process (in MB)").WithDefault(10),
	}
}

func (dlnp *DockerLayerToNP) Process(layer *dockerTypes.DockerLayer) error {
	if layer.Data == nil {
		return fmt.Errorf("layer data is required for NP conversion")
	}

	isConfig, err := dlnp.isConfigDigest(layer)
	if err != nil {
		return fmt.Errorf("failed to determine if digest is config: %w", err)
	}

	if isConfig {
		return dlnp.processConfigData(layer)
	}

	return dlnp.processLayerData(layer)
}

func (dlnp *DockerLayerToNP) isConfigDigest(layer *dockerTypes.DockerLayer) (bool, error) {
	manifest := layer.DockerImage.Manifest
	if manifest == nil {
		return false, fmt.Errorf("manifest required to determine digest type")
	}

	return layer.Digest == manifest.Config.Digest, nil
}

func (dlnp *DockerLayerToNP) processConfigData(layer *dockerTypes.DockerLayer) error {
	npInput := &types.NPInput{
		ContentBase64: base64.StdEncoding.EncodeToString(layer.Data),
		Provenance: types.NPProvenance{
			Platform:     "docker",
			ResourceType: "image",
			ResourceID:   layer.DockerImage.Image,
			Region:       fmt.Sprintf("file:%s", layer.Digest),
		},
	}

	return dlnp.Send(npInput)
}

func (dlnp *DockerLayerToNP) processLayerData(layer *dockerTypes.DockerLayer) error {
	maxFileSizeMB, err := cfg.As[int](dlnp.Arg("max-file-size"))
	if err != nil {
		return err
	}

	sendCb := func(npInput *types.NPInput) error {
		return dlnp.Send(npInput)
	}
	return layer.DockerImage.ProcessLayerWithCallback(bytes.NewReader(layer.Data), layer.Digest, maxFileSizeMB, sendCb)
}
