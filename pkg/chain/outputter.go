package chain

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type Outputter interface {
	outputterMethods
}

type outputterMethods interface {
	cfg.Paramable
	SetLogger(logger *slog.Logger)
	WithLogLevel(level slog.Level) Outputter
	WithLogWriter(w io.Writer) Outputter
	WithLogColoring(color bool) Outputter
	Name() string
	Complete() error
	Initialize() error
}

type BaseOutputter struct {
	*cfg.ContextHolder
	*cfg.ParamHolder
	*cfg.Logger
	name  string
	err   error
	super Outputter
}

func NewBaseOutputter(outputter Outputter, configs ...cfg.Config) *BaseOutputter {
	b := &BaseOutputter{
		ContextHolder: cfg.NewContextHolder(),
		ParamHolder:   cfg.NewParamHolder(),
		Logger:        cfg.NewLogger(),
		name:          fmt.Sprintf("%T", outputter),
		super:         outputter,
	}

	err := b.SetParams(outputter.Params()...)
	if err != nil {
		b.err = err
		return b
	}

	b.WithConfigs(configs...)
	return b
}

func (b *BaseOutputter) Name() string {
	return b.name
}

func (b *BaseOutputter) WithConfigs(configs ...cfg.Config) Outputter {
	for _, config := range configs {
		config(b)
	}
	return b.super
}

func (b *BaseOutputter) SetParams(params ...cfg.Param) error {
	for _, param := range params {
		if err := b.ParamHolder.SetParam(param); err != nil {
			return err
		}
	}
	return nil
}

func (b *BaseOutputter) SetMethods(_ cfg.InjectableMethods) {} // noop - outputters shouldn't have methods

func (o *BaseOutputter) Params() []cfg.Param {
	if o == nil {
		return nil
	}
	return o.ParamHolder.Params()
}

func (o *BaseOutputter) HasParam(name string) bool {
	for _, param := range o.Params() {
		if param.Name() == name {
			return true
		}
	}
	return false
}

func (o *BaseOutputter) Initialize() error {
	return nil
}

func (o *BaseOutputter) Complete() error {
	return nil
}

func (b *BaseOutputter) SetLogger(logger *slog.Logger) {
	b.Logger.SetLogger(logger)
}

func (b *BaseOutputter) WithLogLevel(level slog.Level) Outputter {
	b.Logger.SetLevel(level)
	return b.super
}

func (b *BaseOutputter) WithLogWriter(w io.Writer) Outputter {
	b.Logger.SetWriter(w)
	return b.super
}

func (b *BaseOutputter) WithLogColoring(color bool) Outputter {
	b.Logger.SetColor(color)
	return b.super
}
