package basics

import "github.com/praetorian-inc/janus-framework/pkg/chain"

type MoreThanStrStruct struct {
	Str        string
	Additional string
}

type EvenMoreThanStrStruct struct {
	Str        string
	Additional string
	hidden     string
}

func NewEvenMoreThanStrStruct(str, additional, hidden string) EvenMoreThanStrStruct {
	return EvenMoreThanStrStruct{Str: str, Additional: additional, hidden: hidden}
}

type MoreThanStrStructLink struct {
	*chain.Base
}

func NewMoreThanStrStructLink() chain.Link {
	m := &MoreThanStrStructLink{}
	m.Base = chain.NewBase(m)
	return m
}

func (m *MoreThanStrStructLink) Process(input MoreThanStrStruct) error {
	m.Send(input.Str)
	return nil
}
