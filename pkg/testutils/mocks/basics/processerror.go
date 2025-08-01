package basics

import (
	"errors"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type ProcessErrorLink struct {
	*chain.Base
}

func NewProcessErrorLink(configs ...cfg.Config) chain.Link {
	e := &ProcessErrorLink{}
	e.Base = chain.NewBase(e, configs...)
	return e
}

func (m *ProcessErrorLink) Process(_ string) error {
	return errors.New("mock process error")
}
