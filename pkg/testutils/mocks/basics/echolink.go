package basics

import "github.com/praetorian-inc/janus-framework/pkg/chain"

type EchoLink struct {
	*chain.Base
}

func NewEchoLink() *EchoLink {
	el := &EchoLink{}
	el.Base = chain.NewBase(el)
	return el
}

func (el *EchoLink) Process(item any) error {
	el.Send(item)
	return nil
}
