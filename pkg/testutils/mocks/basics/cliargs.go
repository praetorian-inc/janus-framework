package basics

import (
	"io"
	"os"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type CLIArgsLink struct {
	*chain.Base
	assertFunc func(chain.Link)
}

func NewCLIArgsLink(assertFunc func(chain.Link), configs ...cfg.Config) chain.Link {
	a := &CLIArgsLink{assertFunc: assertFunc}
	a.Base = chain.NewBase(a, configs...)
	return a
}

func (c *CLIArgsLink) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("string", "").WithShortcode("s"),
		cfg.NewParam[string]("stringWithDefault", "").WithShortcode("d").WithDefault("default value"),
		cfg.NewParam[[]string]("stringSlice", "").WithShortcode("slice"),
		cfg.NewParam[[]string]("anotherSlice", "").WithShortcode("anotherslice"),
		cfg.NewParam[int]("int", "").WithShortcode("i"),
		cfg.NewParam[io.Writer]("writer", "").WithShortcode("w").WithConverter(func(s string) (io.Writer, error) {
			return os.Create(s)
		}),
	}
}

func (c *CLIArgsLink) Process(input string) error {
	c.assertFunc(c)

	c.Send(input)
	return nil
}
