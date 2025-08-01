package cfg

type Platform string

const (
	AWS    = Platform("AWS")
	GCP    = Platform("GCP")
	Azure  = Platform("Azure")
	GitHub = Platform("GitHub")
)
