package cfg

import (
	"context"
	"log/slog"
)

type configurable interface {
	Paramable
	SetMethods(methods InjectableMethods)
	Context() context.Context
	SetContext(ctx context.Context)
	SetLogger(logger *slog.Logger)
}

type Config func(configurable) error

func WithArg(name string, value any) Config {
	return func(c configurable) error {
		return c.SetArg(name, value)
	}
}

func WithArgs(args map[string]any) Config {
	return func(c configurable) error {
		for k, v := range args {
			if err := c.SetArg(k, v); err != nil {
				return err
			}
		}
		return nil
	}
}

func WithCLIArgs(args []string) Config {
	return func(c configurable) error {
		return c.SetArgsFromList(args)
	}
}

func WithMethods(methods InjectableMethods) Config {
	return func(c configurable) error {
		c.SetMethods(methods)
		return nil
	}
}

func WithContext(ctx context.Context) Config {
	return func(c configurable) error {
		c.SetContext(ctx)
		return nil
	}
}
