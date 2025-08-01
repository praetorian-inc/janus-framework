package links

import (
	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type AdHocLink[T any] struct {
	*chain.Base
	processFn func(self chain.Link, input T) error
}

func NewAdHocLink[T any](processFn func(self chain.Link, input T) error, configs ...cfg.Config) chain.Link {
	l := &AdHocLink[T]{}
	l.Base = chain.NewBase(l, configs...)
	l.processFn = processFn
	return l
}

func (l *AdHocLink[T]) Process(input T) error {
	return l.processFn(l, input)
}

func ConstructAdHocLink[T any](processFn func(self chain.Link, input T) error) func(...cfg.Config) chain.Link {
	return func(configs ...cfg.Config) chain.Link {
		return NewAdHocLink(processFn, configs...)
	}
}

func FromWrapper[I, O any](wrapper func(I) O, configs ...cfg.Config) chain.Link {
	process := func(self chain.Link, input I) error {
		self.Send(wrapper(input))
		return nil
	}
	return NewAdHocLink(process, configs...)
}

func ConstructWrapper[I, O any](wrapper func(I) O) func(...cfg.Config) chain.Link {
	return func(configs ...cfg.Config) chain.Link {
		return FromWrapper(wrapper, configs...)
	}
}

func FromTransformer[I, O any](transformer func(I) (O, error)) chain.Link {
	process := func(self chain.Link, input I) error {
		output, err := transformer(input)
		if err != nil {
			return err
		}
		self.Send(output)
		return nil
	}
	return NewAdHocLink(process)
}

func FromTransformerSlice[I, O any](transformer func(I) ([]O, error)) chain.Link {
	process := func(self chain.Link, input I) error {
		output, err := transformer(input)
		if err != nil {
			return err
		}
		for _, o := range output {
			self.Send(o)
		}
		return nil
	}
	return NewAdHocLink(process)
}
