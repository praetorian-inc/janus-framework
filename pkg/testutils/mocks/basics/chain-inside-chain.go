package basics

import (
	"fmt"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type ChainInsideChain struct {
	*chain.Base
	prefix string
}

func NewChainInsideChain(configs ...cfg.Config) chain.Link {
	cic := &ChainInsideChain{}
	cic.Base = chain.NewBase(cic, configs...)
	return cic
}

func (c *ChainInsideChain) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("prefix", "Prefix for the inner chain"),
	}
}

func (c *ChainInsideChain) Initialize() error {
	c.prefix = c.Arg("prefix").(string)
	return nil
}

func (c *ChainInsideChain) Process(input string) error {
	inner := chain.NewChain(
		NewStrLink(),
		NewStrIntLink(),
	)

	inner.Send(input)
	inner.Close()

	for output, ok := chain.RecvAs[int](inner); ok; output, ok = chain.RecvAs[int](inner) {
		c.Send(fmt.Sprintf("%s%d", c.prefix, output))
	}

	return inner.Error()
}
