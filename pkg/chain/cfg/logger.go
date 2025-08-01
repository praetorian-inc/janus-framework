package cfg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/lmittmann/tint"
)

var (
	logLock       sync.Mutex
	defaultLevel  slog.Level = slog.LevelInfo
	defaultWriter io.Writer  = os.Stdout
	defaultColor  bool       = false
)

var Levels = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
	"none":  slog.Level(12),
}

func LevelFromString(level string) (slog.Level, error) {
	lvl, ok := Levels[level]
	if !ok {
		return slog.LevelInfo, fmt.Errorf("invalid log level: %s", level)
	}
	return lvl, nil
}

func SetDefaultLevel(level slog.Level) {
	defaultLevel = level
}

func SetDefaultWriter(writer io.Writer) {
	defaultWriter = writer
}

func SetDefaultColor(color bool) {
	defaultColor = color
}

type Logger struct {
	*slog.Logger
	LinkPath string
	Level    slog.Level
	Writer   io.Writer
	Color    bool
}

func NewLogger() *Logger {
	logger := &Logger{LinkPath: "", Level: defaultLevel, Writer: defaultWriter, Color: defaultColor}
	return logger
}

func (l *Logger) SetLinkPath(linkPath string) {
	l.LinkPath = linkPath
}

func (l *Logger) SetWriter(w io.Writer) {
	l.Writer = w
}

func (l *Logger) SetLogger(logger *slog.Logger) {
	l.Logger = logger
}

func (l *Logger) SetLevel(level slog.Level) {
	l.Level = level
}

func (l *Logger) SetColor(color bool) {
	l.Color = color
}

func (l *Logger) Debug(message string, args ...any) {
	if l == nil || l.Logger == nil {
		slog.Error("logger is nil", "message", message, "args", args)
		return
	}
	l.Logger.Debug(message, args...)
}

func (l *Logger) Info(message string, args ...any) {
	if l == nil || l.Logger == nil {
		slog.Error("logger is nil", "message", message, "args", args)
		return
	}
	l.Logger.Info(message, args...)
}

func (l *Logger) Warn(message string, args ...any) {
	if l == nil || l.Logger == nil {
		slog.Error("logger is nil", "message", message, "args", args)
		return
	}
	l.Logger.Warn(message, args...)
}

func (l *Logger) Error(message string, args ...any) {
	if l == nil || l.Logger == nil {
		slog.Error("logger is nil", "message", message, "args", args)
		return
	}
	l.Logger.Error(message, args...)
}

func (l *Logger) Raw(level slog.Level, message string) {
	if l == nil || l.Logger == nil {
		slog.Error("logger is nil", "message", message)
		return
	}

	logLock.Lock()
	defer logLock.Unlock()

	if l.Handler().Enabled(context.Background(), level) {
		fmt.Fprintln(l.Writer, message)
	}
}

func (l *Logger) Initialize() {
	handler := DefaultHandler(&l.LinkPath, l.Writer, &l.Level, &l.Color, nil)
	l.Logger = slog.New(handler)
}

type Handler struct {
	// configurable options
	color    *bool
	level    *slog.Level
	linkPath *string
	writer   io.Writer
	// internal state
	defaultHandler slog.Handler
	intermediate   *bytes.Buffer
}

func DefaultHandler(linkPath *string, w io.Writer, level *slog.Level, color *bool, opts *slog.HandlerOptions) *Handler {
	intermediate := &bytes.Buffer{}

	var defaultHandler slog.Handler
	if color == nil || !*color {
		defaultHandler = slog.NewTextHandler(intermediate, opts)
	} else {
		defaultHandler = tint.NewHandler(intermediate, nil)
	}

	return &Handler{
		color:          color,
		linkPath:       linkPath,
		level:          level,
		writer:         w,
		intermediate:   intermediate,
		defaultHandler: defaultHandler,
	}
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= *h.level
}

func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	h.defaultHandler.Handle(ctx, record)

	message := h.intermediate.String()
	message = h.insertLinkPath(record.Level, message)

	logLock.Lock()
	defer logLock.Unlock()

	_, err := h.writer.Write([]byte(message))
	if err != nil {
		return err
	}

	h.intermediate.Reset()
	return nil
}

func (h *Handler) insertLinkPath(level slog.Level, message string) string {
	if h.linkPath == nil || *h.linkPath == "" {
		return message
	}

	if h.color == nil || !*h.color {
		return strings.Replace(message, "msg=", "link="+*h.linkPath+" msg=", 1)
	}

	switch level {
	case slog.LevelDebug:
		return strings.Replace(message, "DBG", "DBG link="+*h.linkPath, 1)
	case slog.LevelInfo:
		return strings.Replace(message, "INF", "INF link="+*h.linkPath, 1)
	case slog.LevelWarn:
		return strings.Replace(message, "WRN", "WRN link="+*h.linkPath, 1)
	case slog.LevelError:
		return strings.Replace(message, "ERR", "ERR link="+*h.linkPath, 1)
	default:
		return message
	}
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.defaultHandler = h.defaultHandler.WithAttrs(attrs)
	return h
}

func (h *Handler) WithGroup(name string) slog.Handler {
	h.defaultHandler = h.defaultHandler.WithGroup(name)
	return h
}
