package basics

import (
	"errors"
	"regexp"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type ErrorLink struct {
	*chain.Base
	errorAt string
}

func NewErrorLink(configs ...cfg.Config) chain.Link {
	e := &ErrorLink{}
	e.Base = chain.NewBase(e, configs...)
	return e
}

func (l *ErrorLink) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("errorAt", "the function at which the error is returned").
			AsRequired().
			WithRegex(regexp.MustCompile(`^initialize|process|complete$`)),
	}
}

func (l *ErrorLink) Initialize() error {
	l.errorAt, _ = cfg.As[string](l.Arg("errorAt"))

	if l.errorAt == "initialize" {
		return errors.New("initialize error")
	}
	return nil
}

func (l *ErrorLink) Process(v any) error {
	if l.errorAt == "process" {
		return errors.New("process error")
	}
	return nil
}

func (l *ErrorLink) Complete() error {
	if l.errorAt == "complete" {
		return errors.New("complete error")
	}
	return nil
}
