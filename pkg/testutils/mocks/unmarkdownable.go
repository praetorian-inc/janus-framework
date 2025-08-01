package mocks

import (
	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/types"
)

type Unmarkdownable struct {
	ip string
}

func (u *Unmarkdownable) IP() string {
	return u.ip
}

type UnmarkdownableResolver struct {
	*chain.Base
}

func NewUnmarkdownableResolver(configs ...cfg.Config) chain.Link {
	u := &UnmarkdownableResolver{}
	u.Base = chain.NewBase(u, configs...)
	return u
}

func (u *UnmarkdownableResolver) Process(_ types.DomainWrapper) error {
	ip, _ := cfg.As[string](u.Arg("ip"))
	u.Send(&Unmarkdownable{ip: ip})
	return nil
}

func (u *UnmarkdownableResolver) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("ip", "ip address to return").WithDefault("8.8.8.8"),
	}
}
