package mocks

import (
	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/types"
)

type FakeIP struct {
	ip string
}

func NewFakeIP(ip string) *FakeIP {
	return &FakeIP{ip: ip}
}

func (f *FakeIP) IP() string {
	return f.ip
}

type IPResolver struct {
	*chain.Base
}

func NewIPResolver(configs ...cfg.Config) chain.Link {
	i := &IPResolver{}
	i.Base = chain.NewBase(i, configs...)
	return i
}

func (m *IPResolver) Process(input string) error {
	ips, err := cfg.As[[]string](m.Arg("ips"))
	if err != nil {
		return err
	}

	for _, ip := range ips {
		m.Send(types.NewScannableAsset(ip))
	}
	return nil
}

func (m *IPResolver) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[[]string]("ips", "ip addresses to send").WithDefault([]string{"1.1.1.1", "2.2.2.2"}),
	}
}
