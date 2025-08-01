package basics

import (
	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type Collider1 struct {
	*chain.Base
}

func NewCollider1(configs ...cfg.Config) chain.Link {
	c := &Collider1{}
	c.Base = chain.NewBase(c, configs...)
	return c
}

func (c *Collider1) Process(input string) error {
	return nil
}

func (c *Collider1) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("argument", "argument to be collided").AsRequired(),
	}
}

type Collider2 struct {
	*chain.Base
}

func NewCollider2(configs ...cfg.Config) chain.Link {
	c := &Collider2{}
	c.Base = chain.NewBase(c, configs...)
	return c
}

func (c *Collider2) Process(input string) error {
	return nil
}

func (c *Collider2) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[int]("argument", "argument to be collided").AsRequired(),
	}
}
