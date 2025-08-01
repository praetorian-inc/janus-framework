package basics

import (
	"log/slog"
	"strconv"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type StrIntLink struct {
	*chain.Base
	op func(int) int
}

func NewStrIntLink(configs ...cfg.Config) chain.Link {
	s := &StrIntLink{}
	s.Base = chain.NewBase(s, configs...)
	return s
}

func (d *StrIntLink) Process(input string) error {
	slog.Info("StrIntLink.Process", "input", input)
	converted, err := strconv.Atoi(input)
	if err == nil {
		if d.op != nil {
			converted = d.op(converted)
		}
		slog.Info("StrIntLink.Process", "sending", converted)
		d.Send(converted)
	}
	return nil
}
