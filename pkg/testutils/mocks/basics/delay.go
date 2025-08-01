package basics

import (
	"time"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type DelayLink struct {
	*chain.Base
}

func NewDelayLink() *DelayLink {
	dl := &DelayLink{}
	dl.Base = chain.NewBase(dl)
	return dl
}

func (l *DelayLink) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[int]("delay", "number of seconds to wait on each iteration").WithDefault(1),
	}
}

func (l *DelayLink) Process(v any) error {
	duration, err := cfg.As[int](l.Arg("delay"))
	if err != nil {
		return err
	}

	time.Sleep(time.Duration(duration) * time.Second)
	l.Send(v)
	return nil
}
