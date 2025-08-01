package basics

import "github.com/praetorian-inc/janus-framework/pkg/chain"

type StrStructEchoLink struct {
	*chain.Base
}

func NewStrStructEchoLink() chain.Link {
	s := &StrStructEchoLink{}
	s.Base = chain.NewBase(s)
	return s
}

func (s *StrStructEchoLink) Process(input struct{ Str string }) error {
	s.Send(input)
	return nil
}
