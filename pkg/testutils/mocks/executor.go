package mocks

import (
	"fmt"
	"os/exec"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type Executor struct {
	*chain.Base
}

func NewExecutor(configs ...cfg.Config) chain.Link {
	e := &Executor{}
	e.Base = chain.NewBase(e, configs...)
	return e
}

func (e *Executor) Process(_ string) error {
	cmd, err := cfg.As[string](e.Arg("cmd"))
	if err != nil {
		return fmt.Errorf("error retrieving cmd: %v", err)
	}

	args, err := cfg.As[[]string](e.Arg("args"))
	if err != nil {
		return fmt.Errorf("error retrieving args: %v", err)
	}

	parser := func(line string) {
		e.Send(line)
	}

	return e.ExecuteCmd(exec.Command(cmd, args...), parser)
}

func (e *Executor) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("cmd", "command to execute").WithDefault("echo"),
		cfg.NewParam[[]string]("args", "arguments to pass to the command").WithDefault([]string{"hello", "world"}),
	}
}
