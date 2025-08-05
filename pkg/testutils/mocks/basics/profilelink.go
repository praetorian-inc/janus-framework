package basics

import (
	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type ProfileLink struct {
	*chain.Base
}

// NewProfileLink is a test link that declares a profile parameter for testing parameter propagation
func NewProfileLink(configs ...cfg.Config) chain.Link {
	d := &ProfileLink{}
	d.Base = chain.NewBase(d, configs...)
	return d
}

func (d *ProfileLink) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("profile", "profile parameter for testing parameter propagation"),
	}
}

func (d *ProfileLink) Initialize() error {
	return nil
}

func (d *ProfileLink) Process(input string) error {
	d.Send(input)
	return nil
}