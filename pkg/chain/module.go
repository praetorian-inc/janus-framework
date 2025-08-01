package chain

import (
	"fmt"

	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type LinkConstructor func(...cfg.Config) Link

type OutputterConstructor func(...cfg.Config) Outputter

func ConstructLinkWithConfigs(constructor LinkConstructor, configs ...cfg.Config) LinkConstructor {
	return func(args ...cfg.Config) Link {
		return constructor(append(args, configs...)...)
	}
}

func ConstructOutputterWithConfigs(constructor OutputterConstructor, configs ...cfg.Config) OutputterConstructor {
	return func(args ...cfg.Config) Outputter {
		return constructor(append(args, configs...)...)
	}
}

type Module struct {
	metadata     *cfg.Metadata
	constructors []LinkConstructor
	outputters   []OutputterConstructor
	configs      []cfg.Config
	inputParam   cfg.Param
	autoRun      bool
	err          error
}

func NewModule(metadata *cfg.Metadata) *Module {
	return &Module{metadata: metadata}
}

func (m *Module) WithLinks(constructors ...LinkConstructor) *Module {
	m.constructors = constructors
	return m
}

func (m *Module) WithAddedLinks(constructors ...LinkConstructor) *Module {
	m.constructors = append(m.constructors, constructors...)
	return m
}

func (m *Module) WithOutputters(outputters ...OutputterConstructor) *Module {
	m.outputters = outputters
	return m
}

func (m *Module) WithInputParam(param cfg.Param) *Module {
	m.inputParam = param
	return m
}

// WithAutoRun configures the module to automatically run without requiring
// an inputParam value.
func (m *Module) WithAutoRun() *Module {
	m.autoRun = true
	return m
}

func (m *Module) WithConfigs(configs ...cfg.Config) *Module {
	m.configs = configs
	return m
}

func (m *Module) Run(configs ...cfg.Config) error {
	if len(m.outputters) == 0 {
		m.err = fmt.Errorf("module must have outputters to call .Run(). %s has no outputters", m.metadata.Name)
		return m.err
	}

	if m.metadata.InputParam == "" && !m.autoRun {
		m.err = fmt.Errorf("input parameter or AutoRun is required to call .Run(), but module %q has no input parameter", m.metadata.Name)
		return m.err
	}

	c := m.New()
	c.WithConfigs(append(m.configs, configs...)...)
	c.resetParams()

	if !m.autoRun {
		if !c.HasParam(m.metadata.InputParam) {
			m.err = fmt.Errorf("module %q specifies %q as input parameter, but module.Params() does not contain %q", m.metadata.Name, m.metadata.InputParam, m.metadata.InputParam)
			return m.err
		}

		input := c.Arg(m.metadata.InputParam)
		if input == nil {
			m.err = fmt.Errorf("input parameter %q is unset for module %q", m.metadata.InputParam, m.metadata.Name)
			return m.err
		}

		inputSlice, ok := input.([]string)
		if !ok {
			m.err = fmt.Errorf("module input parameter %q must be a slice of strings. %q is a %T", m.metadata.InputParam, input, input)
			return m.err
		}

		for _, input := range inputSlice {
			c.Send(input)
		}
	} else {
		c.Send("autorun")
	}

	c.Close()
	c.Wait()

	m.err = c.Error()
	return m.err
}

func (m *Module) New() Chain {
	links := make([]Link, len(m.constructors))
	for i, constructor := range m.constructors {
		links[i] = constructor()
	}

	outputters := make([]Outputter, len(m.outputters))
	for i, constructor := range m.outputters {
		outputters[i] = constructor()
	}

	c := NewChain(links...).
		WithInputParam(m.inputParam).
		WithOutputters(outputters...).
		WithConfigs(m.configs...)

	m.err = c.Error()
	return c
}

func (m *Module) Metadata() *cfg.Metadata {
	return m.metadata
}

func (m *Module) Params() []cfg.Param {
	return m.New().Params()
}

func (m *Module) Error() error {
	return m.err
}
