package mocks

import (
	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type SuperIP interface {
	IP() string
	Foo() string
}

type SuperIPImpl struct {
	ip string
}

func NewSuperIP(ip string) SuperIP {
	return &SuperIPImpl{ip: ip}
}

func (s *SuperIPImpl) IP() string {
	return s.ip
}

func (s *SuperIPImpl) Foo() string {
	return "bar"
}

type SuperIPLink struct {
	*chain.Base
}

func NewSuperIPLink(configs ...cfg.Config) chain.Link {
	s := &SuperIPLink{}
	s.Base = chain.NewBase(s, configs...)
	return s
}

func (s *SuperIPLink) Process(input string) error {
	s.Send(NewSuperIP(input))
	return nil
}
