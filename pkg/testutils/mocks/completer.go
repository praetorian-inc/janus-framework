package mocks

import "github.com/praetorian-inc/janus-framework/pkg/chain"

type CompleterLink struct {
	*chain.Base
}

func NewCompleterLink() chain.Link {
	c := &CompleterLink{}
	c.Base = chain.NewBase(c)
	return c
}

func (c *CompleterLink) Process(input string) error {
	return nil
}

func (c *CompleterLink) Complete() error {
	c.Send("completed")
	return nil
}
