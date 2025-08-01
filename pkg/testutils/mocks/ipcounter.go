package mocks

import (
	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/types"
)

type IPCounterLink struct {
	*chain.Base
	count int
}

func NewIPCounter(configs ...cfg.Config) chain.Link {
	i := &IPCounterLink{}
	i.Base = chain.NewBase(i, configs...)
	return i
}

func (m *IPCounterLink) Process(input types.IPWrapper) error {
	m.count++
	m.Send(m.count)
	return nil
}
