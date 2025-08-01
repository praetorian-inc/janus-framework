package basics

import (
	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type ArgCheckingLink struct {
	*chain.Base
	assertFunc func(string, error)
}

func NewArgCheckingLink(assertFunc func(string, error), configs ...cfg.Config) chain.Link {
	a := &ArgCheckingLink{assertFunc: assertFunc}
	a.Base = chain.NewBase(a, configs...)
	return a
}

func (a *ArgCheckingLink) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("argument", "test param").WithDefault("default value"),
	}
}

func (a *ArgCheckingLink) Process(input string) error {
	arg, err := cfg.As[string](a.Arg("argument"))
	a.assertFunc(arg, err)

	a.Send(input)
	return nil
}
