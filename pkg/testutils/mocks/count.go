package mocks

import (
	"log/slog"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type CountLink struct {
	*chain.Base
	count int
}

func NewCountLink(configs ...cfg.Config) chain.Link {
	c := &CountLink{}
	c.Base = chain.NewBase(c, configs...)
	return c
}

func (c *CountLink) Process(input any) error {
	c.count++
	slog.Info("CountLink: counting", "item", input)
	c.Send(c.count)
	return nil
}
