package basics

import (
	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type ParamsLink struct {
	*chain.Base
}

func NewParamsLink(configs ...cfg.Config) chain.Link {
	p := &ParamsLink{}
	p.Base = chain.NewBase(p, configs...)
	return p
}

func (p *ParamsLink) Process(input string) error {
	return nil
}

func (p *ParamsLink) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("optional", "optional param"),
		cfg.NewParam[string]("required", "required param").AsRequired(),
		cfg.NewParam[int]("default", "default param").WithDefault(3),
	}
}
