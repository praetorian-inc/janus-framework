package basics

import (
	"fmt"
	"log/slog"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type StrLink struct {
	*chain.Base
	operation func(string) string
}

// NewStrLink accepts string input and returns string output
func NewStrLink(configs ...cfg.Config) chain.Link {
	d := &StrLink{}
	d.Base = chain.NewBase(d, configs...)
	return d
}

func (d *StrLink) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[func(string) string]("strOp", "operation to apply to the input string"),
	}
}

func (d *StrLink) Initialize() error {
	op, ok := d.Arg("strOp").(func(string) string)
	if !ok {
		return fmt.Errorf("strOp must be a func(string) string")
	}

	d.operation = op
	return nil
}

func (d *StrLink) Process(input string) error {
	slog.Debug("StrLink.Process", "input", input)
	if d.operation != nil {
		input = d.operation(input)
		slog.Debug("StrLink.Process", "operation", input)
	}
	d.Send(input)
	slog.Debug("StrLink.Process", "sent", input)
	return nil
}
