package chain

import (
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/praetorian-inc/tabularium/pkg/model/model"

	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cherrors"
)

type Link interface {
	cfg.Paramable
	cfg.InjectableMethods
	withLogWriter(io.Writer) Link
	withLogLevel(slog.Level) Link
	withLogColoring(bool) Link
	AddAncestor(*string) Link
	Name() string
	SetName(string)
	Title() string
	SetTitle(string)
	Initialize() error
	Send(...any) error
	Close()
	Complete() error
	Permissions() []cfg.Permission
	Invoke(...any) ([]any, error)
	children() []Link
	channel() chan any
	isClaimed() bool
	isChain() bool
	Error() error
	SetError(error)
	CredentialType() model.CredentialType
	claim()
	// start is the main entry point for the link. It must be called from a goroutine.
	start(chan any, func(error), Strictness)
}

type Base struct {
	*cfg.ContextHolder
	*cfg.ParamHolder
	*cfg.MethodsHolder
	Logger      *cfg.Logger
	linkPath    []*string
	name        string
	title       string
	ch          chan any
	closeOnce   sync.Once
	super       Link
	strictness  Strictness
	err         error
	claimed     bool
	permissions []cfg.Permission
}

func NewBase(link Link, configs ...cfg.Config) *Base {
	if link == nil {
		panic("link is <nil> in call to NewBase()")
	}

	linkName := fmt.Sprintf("%T", link)

	b := &Base{
		ContextHolder: cfg.NewContextHolder(),
		ParamHolder:   cfg.NewParamHolder(),
		MethodsHolder: cfg.NewMethodsHolder(),
		Logger:        cfg.NewLogger(),
		ch:            make(chan any),
		super:         link,
		name:          linkName,
	}
	b.linkPath = []*string{&b.name}

	b.initializeLogger()

	err := b.SetParams(link.Params()...)
	if err != nil {
		b.err = err
		return b
	}

	return b.WithConfigs(configs...)
}

func (b *Base) Name() string {
	if b == nil {
		return ""
	}
	return b.name
}

func (b *Base) SetName(name string) {
	b.name = name
	b.initializeLogger()
}

func (b *Base) Title() string {
	if b == nil {
		return ""
	}
	if b.title == "" {
		return b.name
	}
	return b.title
}

func (b *Base) SetTitle(title string) {
	b.title = title
	b.initializeLogger()
}

func (b *Base) Initialize() error {
	return nil
}

func (b *Base) Complete() error {
	return nil
}

func (b *Base) WithConfigs(configs ...cfg.Config) *Base {
	for _, config := range configs {
		if err := config(b); err != nil {
			b.err = err
			return b
		}
	}
	return b
}

func (b *Base) CredentialType() model.CredentialType {
	return ""
}

func (b *Base) Params() []cfg.Param {
	if b == nil {
		return nil
	}
	return b.ParamHolder.Params()
}

func (b *Base) SetParams(params ...cfg.Param) error {
	for _, param := range params {
		err := b.SetParam(param)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Base) HasParam(name string) bool {
	for _, param := range b.Params() {
		if param.Name() == name {
			return true
		}
	}
	return false
}

func (b *Base) Permissions() []cfg.Permission {
	if b == nil {
		return nil
	}
	return b.permissions
}

func (b *Base) Send(values ...any) error {
	for _, v := range values {
		b.ch <- v
	}
	return nil
}

func (b *Base) children() []Link {
	return []Link{}
}

func (b *Base) start(prevChannel chan any, errHandler func(error), strictness Strictness) {
	defer func() {
		err := b.cleanup(errHandler)
		if err != nil {
			err = fmt.Errorf("failed to cleanup link: %w", err)
			errHandler(err)
		}
	}()

	b.initializeLogger()

	err := b.initialize(errHandler)
	if err != nil {
		errHandler(err)
		return
	}

	pc := prevChannel
	b.processLoop(pc, errHandler, strictness)
}

func (b *Base) initialize(errHandler func(error)) error {
	err := b.super.Initialize()
	if err != nil {
		err = fmt.Errorf("link %s failed to initialize: %v", b.Name(), err)
		errHandler(err)
	}

	err = b.ParamHolder.Validate()
	if err != nil {
		err = fmt.Errorf("link %s failed to validate params: %v", b.Name(), err)
		errHandler(err)
	}

	return err
}

func (b *Base) cleanup(errHandler func(error)) error {
	err := b.super.Complete()
	if err != nil {
		err = fmt.Errorf("failed to complete link: %w", err)
		errHandler(err)
	}
	b.Close()
	return err
}

func (b *Base) processLoop(prevChannel chan any, errHandler func(error), strictness Strictness) {
	b.strictness = strictness
	ignoreRemaining := false
	for v := range prevChannel {
		if ignoreRemaining { // necessary to prevent deadlock from earlier chains
			continue
		}

		err := b.process(v, errHandler)
		if err != nil {
			ignoreRemaining = true
		}
	}
}

func (b *Base) process(v any, errHandler func(error)) error {
	err := Process(b.super, v)
	if b.shouldBreak(err) {
		err = fmt.Errorf("link encountered error, killing chain due to strictness (%s): %w", b.strictness.String(), err)
		errHandler(err)
		return err
	}
	return nil
}

func (b *Base) shouldBreak(err error) bool {
	if err == nil {
		return false
	}

	if b.handleDebugError(err) {
		return false
	}

	_, isConversionError := err.(*cherrors.ConversionError)
	b.logLinkError(err, isConversionError)

	if b.strictness == Lax {
		return false
	}

	if b.strictness == Moderate && isConversionError {
		return false
	}

	return true
}

func (b *Base) handleDebugError(err error) bool {
	_, isDebugError := err.(*cherrors.DebugError)
	if isDebugError {
		b.Logger.Debug("encountered debug error, continuing", "error", err)
	}
	return isDebugError
}

func (b *Base) logLinkError(err error, isConversionError bool) {
	logMsg := "process error"
	if isConversionError {
		logMsg = "conversion error"
	}

	b.Logger.Error(logMsg, "error", err)
}

func (b *Base) Close() {
	b.closeOnce.Do(func() {
		close(b.ch)
	})
}

func (b *Base) channel() chan any {
	return b.ch
}

func (b *Base) Invoke(input ...any) ([]any, error) {
	if b.super.Initialize() != nil {
		return nil, fmt.Errorf("link %s failed to initialize", b.Name())
	}

	b.initializeLogger()

	output := []any{}
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for v := range b.channel() {
			output = append(output, v)
		}
	}()

	for _, item := range input {
		err := Process(b.super, item)

		if b.handleDebugError(err) {
			continue
		}

		if err != nil {
			b.Close()
			return nil, err
		}
	}

	b.Close()
	wg.Wait()

	return output, nil
}

func (b *Base) isClaimed() bool {
	if b == nil {
		return false
	}
	return b.claimed
}

func (b *Base) isChain() bool {
	return false
}

func (b *Base) claim() {
	b.claimed = true
}

func (b *Base) Error() error {
	return b.err
}

func (b *Base) SetError(err error) {
	if b != nil {
		b.err = err
	}
}

func (b *Base) SetLogger(logger *slog.Logger) {
	b.Logger.SetLogger(logger)
}

func (b *Base) withLogLevel(level slog.Level) Link {
	b.Logger.SetLevel(level)
	return b.super
}

func (b *Base) withLogWriter(w io.Writer) Link {
	b.Logger.SetWriter(w)
	return b.super
}

func (b *Base) withLogColoring(color bool) Link {
	b.Logger.SetColor(color)
	return b.super
}

func (b *Base) AddAncestor(name *string) Link {
	b.linkPath = append(b.linkPath, name)
	return b.super
}

func (b *Base) LinkPath() string {
	if b == nil {
		return ""
	}
	path := *b.linkPath[0]
	for i := 1; i < len(b.linkPath); i++ {
		path = *b.linkPath[i] + "/" + path
	}

	return path
}

func (b *Base) initializeLogger() {
	b.Logger.SetLinkPath(b.LinkPath())
	b.Logger.Initialize()
}
