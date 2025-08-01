package links

import (
	"fmt"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
)

type StringConverter struct {
	*chain.Base
	next chain.Link
}

func NewStringConverter(next chain.Link) *StringConverter {
	c := &StringConverter{}
	c.Base = chain.NewBase(c)
	c.next = next
	return c
}

func (c *StringConverter) Process(input string) error {
	converted, err := chain.ConvertForLink(input, c.next)
	if err == nil {
		return c.Send(converted)
	}

	converted, err = chain.ConvertForJSON(input, c.next)
	if err == nil {
		return c.Send(converted)
	}

	return fmt.Errorf("failed to convert input to %T", c.next)
}
