package mocks

import (
	"errors"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type ErrorLink struct {
	*chain.Base
}

func NewErrorLink(configs ...cfg.Config) chain.Link {
	e := &ErrorLink{}
	e.Base = chain.NewBase(e, configs...)
	return e
}

func (m *ErrorLink) Validate() error {
	return errors.New("mock error")
}

func (m *ErrorLink) Process(input string) error {
	m.Send(input)
	return nil
}

func (m *ErrorLink) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("error", "error to return from Validate(), defaults to 'mock error'").WithDefault("mock error"),
	}
}
