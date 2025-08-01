package mocks

import (
	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/types"
)

type EchoIPLink struct {
	*chain.Base
}

func NewEchoIPLink(configs ...cfg.Config) chain.Link {
	e := &EchoIPLink{}
	e.Base = chain.NewBase(e, configs...)
	return e
}

func (e *EchoIPLink) Process(input *FakeIP) error {
	e.Send(types.NewScannableAsset(input.IP()))
	return nil
}

type EchoStringLink struct {
	*chain.Base
}

func NewEchoStringLink(configs ...cfg.Config) chain.Link {
	e := &EchoStringLink{}
	e.Base = chain.NewBase(e, configs...)
	return e
}

func (e *EchoStringLink) Process(input string) error {
	prefix, _ := cfg.As[string](e.Arg("prefix"))
	e.Send(prefix + input)
	return nil
}

func (e *EchoStringLink) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("prefix", "prefix to append to the input"),
	}
}
