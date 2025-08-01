package cfg_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	w := &bytes.Buffer{}
	logger := cfg.NewLogger()
	logger.SetLinkPath("MyLink")
	logger.SetLevel(slog.LevelInfo)
	logger.SetWriter(w)
	logger.SetColor(false)
	logger.Initialize()

	logger.Info("test")

	assert.Contains(t, w.String(), "level=INFO link=MyLink msg=test")
}

func TestLogger_DefaultHandler(t *testing.T) {
	w := &bytes.Buffer{}
	linkPath := "MyLink"
	level := slog.LevelInfo
	color := false
	handler := cfg.DefaultHandler(&linkPath, w, &level, &color, nil)

	handler.Handle(context.Background(), slog.Record{Level: slog.LevelInfo, Message: "test"})
	assert.Contains(t, w.String(), "level=INFO link=MyLink msg=test")

	logger := slog.New(handler)
	logger.Info("second test")

	assert.Contains(t, w.String(), "level=INFO link=MyLink msg=\"second test\"")
}

func TestLogger_SetLevel(t *testing.T) {
	w := &bytes.Buffer{}
	logger := cfg.NewLogger()
	logger.SetLinkPath("MyLink")
	logger.SetLevel(slog.LevelDebug)
	logger.SetWriter(w)
	logger.SetColor(false)
	logger.Initialize()

	logger.Debug("Debug Message")
	logger.SetLevel(slog.LevelInfo)
	logger.Debug("Debug Message Again")

	assert.Contains(t, w.String(), "level=DEBUG link=MyLink msg=\"Debug Message\"")
	assert.NotContains(t, w.String(), "level=INFO link=MyLink msg=\"Debug Message Again\"")
}

func TestLogger_SetDefaultLevel(t *testing.T) {
	cfg.SetDefaultLevel(slog.LevelWarn)
	t.Cleanup(func() {
		cfg.SetDefaultLevel(slog.LevelInfo)
	})

	w := &bytes.Buffer{}
	logger := cfg.NewLogger()
	logger.SetWriter(w)
	logger.SetColor(false)
	logger.Initialize()

	logger.Debug("Debug Message")
	logger.Info("Info Message")
	logger.Warn("Warn Message")
	logger.Error("Error Message")

	formatted := strings.ReplaceAll(w.String(), "\\n", "\n")

	assert.Contains(t, w.String(), "level=WARN msg=\"Warn Message\"", formatted)
	assert.Contains(t, w.String(), "level=ERROR msg=\"Error Message\"", formatted)
	assert.NotContains(t, w.String(), "level=DEBUG msg=\"Debug Message\"", formatted)
	assert.NotContains(t, w.String(), "level=INFO msg=\"Info Message\"", formatted)
}

func TestLogger_SetDefaultWriter(t *testing.T) {
	globalBuf := &bytes.Buffer{}
	cfg.SetDefaultWriter(globalBuf)
	t.Cleanup(func() {
		cfg.SetDefaultWriter(os.Stdout)
	})

	logger := cfg.NewLogger()
	logger.SetColor(false)
	logger.Initialize()

	logger.Info("Info Message")

	assert.Contains(t, globalBuf.String(), "level=INFO msg=\"Info Message\"")

	newBuf := &bytes.Buffer{}
	logger.SetWriter(newBuf)
	logger.Initialize()

	logger.Info("New Message")

	assert.Contains(t, newBuf.String(), "level=INFO msg=\"New Message\"")
	assert.NotContains(t, globalBuf.String(), "level=INFO msg=\"New Message\"")
}

func TestLogger_SetColor(t *testing.T) {
	w := &bytes.Buffer{}

	logger := cfg.NewLogger()
	logger.SetColor(true)
	logger.SetWriter(w)
	logger.Initialize()

	logger.Info("Info Message")

	assert.Contains(t, w.String(), "\x1b[92mINF\x1b[0m Info Message")
}

func TestLogger_SetDefaultColor(t *testing.T) {
	w := &bytes.Buffer{}
	cfg.SetDefaultColor(true)
	t.Cleanup(func() {
		cfg.SetDefaultColor(false)
	})

	logger := cfg.NewLogger()
	logger.SetWriter(w)
	logger.Initialize()

	logger.Info("Info Message")

	assert.Contains(t, w.String(), "\x1b[92mINF\x1b[0m Info Message")
}
