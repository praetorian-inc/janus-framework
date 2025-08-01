package mocks

import (
	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type ContextLink struct {
	*chain.Base
}

func NewContextLink(configs ...cfg.Config) chain.Link {
	c := &ContextLink{}
	c.Base = chain.NewBase(c, configs...)
	return c
}

func (m *ContextLink) Process(_ string) error {
	fooVal := m.Context().Value("foo")
	m.Send(fooVal.(string))
	return nil
}
