package basics

import "github.com/praetorian-inc/janus-framework/pkg/chain"

type MockerEchoLink struct {
	*chain.Base
}

func NewMockerEchoLink() chain.Link {
	m := &MockerEchoLink{}
	m.Base = chain.NewBase(m)
	return m
}

func (m *MockerEchoLink) Process(input Mocker) error {
	m.Send(input)
	return nil
}
