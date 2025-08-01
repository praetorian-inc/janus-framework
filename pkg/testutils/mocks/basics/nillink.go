package basics

import "github.com/praetorian-inc/janus-framework/pkg/chain"

type NilLink struct {
	*chain.Base
}

func NewNilLink() *NilLink {
	return &NilLink{
		Base: chain.NewBase(nil),
	}
}

func (l *NilLink) Process(v any) error {
	return nil
}
