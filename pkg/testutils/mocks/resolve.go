package mocks

import (
	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/types"
)

type ResolveLink struct {
	*chain.Base
}

func NewResolve(configs ...cfg.Config) chain.Link {
	r := &ResolveLink{}
	r.Base = chain.NewBase(r, configs...)
	return r
}

func (m *ResolveLink) Process(_ types.IPWrapper) error {
	ips, err := cfg.As[[]string](m.Arg("ips"))
	if err != nil {
		return err
	}

	for _, ip := range ips {
		m.Send(types.NewScannableAsset(ip))
	}
	return nil
}

func (m *ResolveLink) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[[]string]("ips", "mock ips to return").WithDefault([]string{"1.1.1.1", "2.2.2.2"}),
	}
}
