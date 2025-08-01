package basics

import (
	"fmt"
	"log/slog"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type Msg struct {
	Level   slog.Level
	Message string
}

type LoggingLink struct {
	*chain.Base
}

func NewLoggingLink(configs ...cfg.Config) chain.Link {
	ll := &LoggingLink{}
	ll.Base = chain.NewBase(ll, configs...)
	return ll
}

func (ll *LoggingLink) Process(input Msg) error {
	switch input.Level {
	case slog.LevelDebug:
		ll.Logger.Debug(input.Message)
	case slog.LevelInfo:
		ll.Logger.Info(input.Message)
	case slog.LevelWarn:
		ll.Logger.Warn(input.Message)
	case slog.LevelError:
		ll.Logger.Error(input.Message)
	default:
		return fmt.Errorf("unknown log level: %s", input.Level)
	}
	return nil
}
