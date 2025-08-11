package docker

type RegistryAuthResponse struct {
	Token string `json:"token"`
}

type RegistryManifestV2 struct {
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
