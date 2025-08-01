package output

import (
	"fmt"
	"io"
	"os"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type WriterOutputter struct {
	*chain.BaseOutputter
	writer io.Writer
}

func NewWriterOutputter(configs ...cfg.Config) chain.Outputter {
	o := &WriterOutputter{}
	o.BaseOutputter = chain.NewBaseOutputter(o, configs...)
	return o
}

func NewConsoleOutputter(configs ...cfg.Config) chain.Outputter {
	o := &WriterOutputter{}
	o.BaseOutputter = chain.NewBaseOutputter(o, configs...)
	o.writer = os.Stdout
	return o
}

func (o *WriterOutputter) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[io.Writer]("writer", "io.Writer to write to").WithDefault(os.Stdout),
	}
}

func (o *WriterOutputter) Initialize() error {
	writer, err := cfg.As[io.Writer](o.Arg("writer"))
	if err != nil {
		return fmt.Errorf("writer is not an io.Writer: %w", err)
	}

	o.writer = writer
	return nil
}

func (o *WriterOutputter) Output(v any) error {
	_, err := fmt.Fprintln(o.writer, v)
	return err
}
