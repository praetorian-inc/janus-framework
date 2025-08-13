package docker

import (
	"crypto/sha256"
	"fmt"
)

type RegistryAuthResponse struct {
	Token string `json:"token"`
}

type RegistryLayer struct {
    MediaType string `json:"mediaType"`
    Digest    string `json:"digest"`
}

// createIdFromParent creates a unique, deterministic ID for the layer based on its parent ID and digest
func (l *RegistryLayer) createIdFromParent(parentId string) string {
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
