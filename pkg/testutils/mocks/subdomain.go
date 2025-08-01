package mocks

import (
	"fmt"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/types"
)

type SubdomainLink struct {
	*chain.Base
}

func NewSubdomain(configs ...cfg.Config) chain.Link {
	s := &SubdomainLink{}
	s.Base = chain.NewBase(s, configs...)
	return s
}

func (m *SubdomainLink) Process(dw types.DomainWrapper) error {
	domain := dw.Domain

	subdomains, err := cfg.As[[]string](m.Arg("subdomains"))
	if err != nil {
		return err
	}

	for _, sub := range subdomains {
		m.Send(types.NewScannableAsset(fmt.Sprintf("%s.%s", sub, domain)))
	}
	return nil
}

func (m *SubdomainLink) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[[]string]("subdomains", "mock subdomains to return").WithDefault([]string{"sub1", "sub2", "sub3"}),
	}
}
