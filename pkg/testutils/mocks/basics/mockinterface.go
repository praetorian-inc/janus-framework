package basics

import "github.com/praetorian-inc/janus-framework/pkg/chain"

type Mocker interface {
	Mock() string
}

type Mockable struct {
	MockMsg string
}

func (m *Mockable) Mock() string {
	return m.MockMsg
}

type StructMockable struct {
	Msg string
}

func (m StructMockable) Mock() string {
	return m.Msg
}

type interfaceLink struct {
	*chain.Base
}

func NewInterfaceLink() chain.Link {
	i := &interfaceLink{}
	i.Base = chain.NewBase(i)
	return i
}

func (i *interfaceLink) Process(input Mocker) error {
	i.Send(input.Mock())
	return nil
}
