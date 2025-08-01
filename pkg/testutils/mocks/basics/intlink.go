package basics

import (
	"fmt"
	"log/slog"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type IntLink struct {
	*chain.Base
	operation func(int) int
}

func NewIntLink(configs ...cfg.Config) chain.Link {
	i := &IntLink{}
	i.Base = chain.NewBase(i, configs...)
	return i
}

func (d *IntLink) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[func(int) int]("intOp", "operation to apply to the input int"),
	}
}

func (d *IntLink) Initialize() error {
	op, ok := d.Arg("intOp").(func(int) int)
	if !ok {
		return fmt.Errorf("intOp must be a func(int) int")
	}
	d.operation = op
	return nil
}

func (d *IntLink) Process(input int) error {
	slog.Debug("IntLink.Process", "input", input)
	if d.operation != nil {
		input = d.operation(input)
		slog.Debug("IntLink.Process", "operation", input)
	}
	d.Send(input)
	return nil
}
