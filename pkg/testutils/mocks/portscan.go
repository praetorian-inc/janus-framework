package mocks

import (
	"fmt"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/types"
)

type PortscanLink struct {
	*chain.Base
	index int
}

func NewPortscan(configs ...cfg.Config) chain.Link {
	p := &PortscanLink{}
	p.Base = chain.NewBase(p, configs...)
	return p
}

func (m *PortscanLink) Process(input types.ScannableAsset) error {
	ports, err := cfg.As[[]string](m.Arg("ports"))
	if err != nil {
		return err
	}

	m.Send(fmt.Sprintf("%s:%s", input.IP, ports[m.index]))
	m.index = (m.index + 1) % len(ports)

	return nil
}

func (m *PortscanLink) Params() []cfg.Param {
	ports := []string{"80", "443", "22", "21", "25", "53", "110", "143", "389", "465", "587", "993", "995"}

	return []cfg.Param{
		cfg.NewParam[[]string]("ports", "mock ports to return").WithDefault(ports),
	}
}
