package links

import (
	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type Count struct {
	*chain.Base
	count int
}

func NewCount(configs ...cfg.Config) chain.Link {
	c := &Count{}
	c.Base = chain.NewBase(c, configs...)
	return c
}

func (c *Count) Process(resource string) error {
	c.count++
	c.Send(c.count)
	return nil
}
