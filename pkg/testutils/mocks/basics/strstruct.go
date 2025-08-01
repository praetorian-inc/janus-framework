package basics

import "github.com/praetorian-inc/janus-framework/pkg/chain"

type StrStructLink struct {
	*chain.Base
}

func NewStrStructLink() chain.Link {
	s := &StrStructLink{}
	s.Base = chain.NewBase(s)
	return s
}

func (s *StrStructLink) Process(input struct{ Str string }) error {
	s.Send(input.Str)
	return nil
}
