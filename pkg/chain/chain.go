package chain

import (
	"fmt"
	"io"
	"log/slog"
	"maps"
	"sync"

	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/util"
)

// Chain is a collection of Links and Outputters that can be used to process data.
//
// Chains call .Send() to pass data to the first Link in the chain. After the caller has finished calling .Send(),
// it should call .Close() to close the chain. The caller can use .Error() to check if the chain encountered an error.
//
// If the Chain has outputters, the caller should use .Wait() to wait for the chain to finish processing.
//
// If the Chain does not have outputters, the caller should use RecvAs() to retrieve one element at a time from the chain.
type Chain interface {
	Link
	WithConfigs(configs ...cfg.Config) Chain
	WithOutputters(outputters ...Outputter) Chain
	WithStrictness(strictness Strictness) Chain
	WithInputParam(param cfg.Param) Chain
	WithName(name string) Chain
	WithAddedLinks(links ...Link) Chain
	WithLogLevel(level slog.Level) Chain
	WithLogWriter(w io.Writer) Chain
	WithLogColoring(color bool) Chain
	// Waits for the chain to finish processing. Will discard all output if there are no outputters configured.
	Wait()
	// Closes the chain. Links will process any remaining data, and then close themselves.
	Close()
	// Error returns the first error reported by a link in the chain. If no link has reported an error yet, Error() will return nil.
	Error() error
	Outputters() []Outputter
	PermissionsMap() map[cfg.Platform][]string
	resetParams() error
}

type BaseChain struct {
	links        []Link
	started      bool
	startLock    sync.Mutex
	wgOut        sync.WaitGroup
	outputters   []Outputter
	chanIn       chan any
	closeChanIn  sync.Once
	errLock      sync.Mutex
	super        Chain // TODO: do I need this?
	outputItems  []any
	isClosed     bool
	addedConfigs []cfg.Config
	inputParam   cfg.Param
	*Base
}

func NewChain(links ...Link) Chain {
	c := &BaseChain{links: links, chanIn: make(chan any)}
	c.Base = NewBase(c)
	c.super = c

	for _, link := range links {
		if link.isClaimed() {
			c.handleError(fmt.Errorf("link %s is in-use by another chain", link.Name()))
		}
		link.claim()
		if link.Error() != nil {
			c.handleError(link.Error())
		}
		link.AddAncestor(&c.Base.name)
	}

	return c
}

func (c *BaseChain) isChain() bool {
	return true
}

func (c *BaseChain) WithConfigs(configs ...cfg.Config) Chain {
	c.addedConfigs = configs
	return c.super
}

func (c *BaseChain) WithOutputters(outputters ...Outputter) Chain {
	c.outputters = outputters
	return c.super
}

func (c *BaseChain) WithStrictness(strictness Strictness) Chain {
	c.Base.strictness = strictness
	return c.super
}

func (c *BaseChain) WithInputParam(param cfg.Param) Chain {
	if param != nil {
		c.inputParam = param
	}
	return c.super
}

func (c *BaseChain) WithName(name string) Chain {
	c.Base.name = name
	return c.super
}

func (c *BaseChain) WithLogWriter(w io.Writer) Chain {
	c.withLogWriter(w)
	return c.super
}

func (c *BaseChain) withLogWriter(w io.Writer) Link {
	c.Base.withLogWriter(w)
	for _, link := range c.children() {
		link.withLogWriter(w)
	}
	return c.super
}

func (c *BaseChain) WithLogLevel(level slog.Level) Chain {
	c.withLogLevel(level)
	return c.super
}

func (c *BaseChain) withLogLevel(level slog.Level) Link {
	c.Base.withLogLevel(level)
	for _, link := range c.children() {
		link.withLogLevel(level)
	}
	return c.super
}

func (c *BaseChain) WithLogColoring(color bool) Chain {
	c.withLogColoring(color)
	return c.super
}

func (c *BaseChain) withLogColoring(color bool) Link {
	c.Base.withLogColoring(color)
	for _, link := range c.children() {
		link.withLogColoring(color)
	}
	return c.super
}

func (c *BaseChain) AddAncestor(name *string) Link {
	c.Base.AddAncestor(name)
	for _, link := range c.children() {
		link.AddAncestor(name)
	}
	return c.super
}

func (c *BaseChain) WithAddedLinks(links ...Link) Chain {
	c.links = append(c.links, links...)
	return c.super
}

func (c *BaseChain) Start() error {
	if c.inputParam == nil {
		err := fmt.Errorf("chain has no input param")
		c.handleError(err)
		return err
	}

	arg := c.Arg(c.inputParam.Name())
	if arg == nil {
		err := fmt.Errorf("chain input param %q has no value", c.inputParam.Name())
		c.handleError(err)
		return err
	}

	c.Send(arg)
	c.Close()

	return nil
}

func (c *BaseChain) Send(values ...any) error {
	if c.isClosed {
		err := fmt.Errorf("chain is closed")
		c.handleError(err)
		return err
	}

	c.startIfUnstarted()
	c.errLock.Lock()
	defer c.errLock.Unlock()

	if err := c.getError(); err != nil {
		return fmt.Errorf("chain is in error state: %w", err)
	}

	for _, v := range values {
		c.chanIn <- v
	}

	return nil
}

func (c *BaseChain) Process(v any) error {
	c.links[0].channel() <- v
	return nil
}

func (c *BaseChain) Close() {
	c.startIfUnstarted()
	c.closeChanInOnce()
	c.isClosed = true
}

func (c *BaseChain) Wait() {
	c.startIfUnstarted()

	c.errLock.Lock()
	err := c.getError()
	c.errLock.Unlock()

	if err != nil {
		return
	}

	if len(c.outputters) == 0 {
		// caller has called Wait() with no outputters.
		// c.wgOut.Wait() will deadlock if we do not empty this channel.
		util.EmptyChannel(c.channel())
	}

	c.wgOut.Wait()
}

func (c *BaseChain) Error() error {
	return c.getError()
}

func (c *BaseChain) startIfUnstarted() {
	if !c.hasStarted() {
		c.start(c.chanIn, c.handleError, c.strictness)
	}
	c.setStarted()
}

func (c *BaseChain) hasStarted() bool {
	c.startLock.Lock()
	defer c.startLock.Unlock()

	return c.started
}

func (c *BaseChain) setStarted() {
	c.startLock.Lock()
	defer c.startLock.Unlock()

	c.started = true
}

func (c *BaseChain) start(prevChan chan any, errHandler func(error), strictness Strictness) {
	c.initializeLogger()

	if err := c.resetParams(); err != nil {
		errHandler(err)
		return
	}

	for _, outputter := range c.outputters {
		if err := c.startOutputter(outputter); err != nil {
			errHandler(err)
		}
	}

	for _, child := range c.children() {
		prevChan = c.startChild(child, prevChan, errHandler, strictness)
	}

	c.wgOut.Add(1)
	go c.collectOutput(prevChan, errHandler)
}

func (c *BaseChain) resetParams() error {
	err := c.Base.SetParams(c.Params()...)
	if err != nil {
		return err
	}

	c.Base.WithConfigs(c.addedConfigs...)
	return nil
}

func (c *BaseChain) startOutputter(outputter Outputter) error {
	err := c.setAllArgs(outputter)
	if err != nil {
		return err
	}

	err = outputter.Initialize()
	if err != nil {
		return err
	}

	return nil
}

func (c *BaseChain) startChild(child Link, prevChan chan any, errHandler func(error), strictness Strictness) chan any {
	if err := c.setArgs(child); err != nil {
		errHandler(err)
		return nil
	}

	go child.start(prevChan, errHandler, strictness)
	return child.channel()
}

func (c *BaseChain) setArgs(paramable cfg.Paramable) error {
	for key, arg := range c.Args() {
		expectsParam := paramable.HasParam(key)
		hasArg := paramable.WasSet(key)

		if expectsParam && !hasArg {
			if err := paramable.SetArg(key, arg); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *BaseChain) setAllArgs(paramable cfg.Paramable) error {
	allArgs := make(map[string]any)

	maps.Copy(allArgs, c.Args())

	for _, link := range c.children() {
		for key, arg := range link.Args() {
			if _, exists := allArgs[key]; !exists {
				allArgs[key] = arg
			}
		}
	}

	// For outputters, dynamically declare parameters that don't exist
	for key, arg := range allArgs {
		if !paramable.HasParam(key) {
			// Create parameter dynamically for this outputter based on the argument type
			switch arg.(type) {
			case string:
				param := cfg.NewParam[string](key, "dynamically propagated parameter from chain/links")
				if err := paramable.SetParams(param); err != nil {
					return err
				}
			case int:
				param := cfg.NewParam[int](key, "dynamically propagated parameter from chain/links")
				if err := paramable.SetParams(param); err != nil {
					return err
				}
			default:
				param := cfg.NewParam[any](key, "dynamically propagated parameter from chain/links")
				if err := paramable.SetParams(param); err != nil {
					return err
				}
			}
		}

		if err := paramable.SetArg(key, arg); err != nil {
			return err
		}
	}
	return nil
}

func (c *BaseChain) getError() error {
	return c.err
}

func (c *BaseChain) handleError(err error) {
	if c.hasStarted() {
		util.EmptyChannel(c.chanIn)
	}

	c.errLock.Lock()
	defer c.errLock.Unlock()

	c.closeChanInOnce()
	if c.err != nil {
		return
	}

	c.err = err
}

func (c *BaseChain) closeChanInOnce() {
	c.closeChanIn.Do(func() {
		close(c.chanIn)
	})
}

func (c *BaseChain) collectOutput(lastLinkChan chan any, errHandler func(error)) {
	defer func() {
		c.flushOutputItems()
		close(c.channel())
		if err := c.closeOutputters(); err != nil {
			errHandler(err)
		}
		c.wgOut.Done()
	}()

	for input := range lastLinkChan {
		if err := c.output(input); err != nil {
			errHandler(err)
		}
	}
}

func (c *BaseChain) output(value any) error {
	if len(c.outputters) == 0 {
		return c.outputToSelf(value)
	}

	return c.outputToOutputters(value)
}

func (c *BaseChain) outputToSelf(value any) error {
	converted, err := ConvertForLink(value, c)
	if err != nil {
		return fmt.Errorf("chain collector failed to convert item: %w", err)
	}
	c.outputItems = append(c.outputItems, converted)
	return nil
}

func (c *BaseChain) outputToOutputters(value any) error {
	for _, outputter := range c.outputters {
		err := Output(outputter, value)
		if err != nil {
			c.Logger.Warn(fmt.Sprintf("chain outputter %T failed to output item", outputter), "item", value, "error", err)
		}
	}
	return nil
}

func (c *BaseChain) flushOutputItems() error {
	for _, item := range c.outputItems {
		c.channel() <- item
	}
	c.outputItems = nil
	return nil
}

func (c *BaseChain) closeOutputters() error {
	for _, outputter := range c.outputters {
		if err := outputter.Complete(); err != nil {
			return err
		}
	}
	return nil
}

func (c *BaseChain) Params() []cfg.Param {
	params := []cfg.Param{}
	if c.inputParam != nil {
		params = append(params, c.inputParam)
	}

	seen := make(map[string]bool)
	paramables := c.paramables()

	for _, paramable := range paramables {
		for _, param := range paramable.Params() {
			if !seen[param.Identifier()] {
				params = append(params, param)
				seen[param.Identifier()] = true
			}
		}
	}

	return params
}

func (c *BaseChain) paramables() []cfg.Paramable {
	paramables := []cfg.Paramable{}
	// paramables = append(paramables, c.Base)

	for _, link := range c.children() {
		paramables = append(paramables, link)
	}
	for _, outputter := range c.outputters {
		paramables = append(paramables, outputter)
	}
	return paramables
}

func (c *BaseChain) HasParam(name string) bool {
	for _, param := range c.Params() {
		if param.Name() == name {
			return true
		}
	}
	for _, link := range c.children() {
		if link.HasParam(name) {
			return true
		}
	}
	return false
}

func (c *BaseChain) SetArg(name string, value any) error {
	for _, link := range c.children() {
		if link.HasParam(name) {
			if err := link.SetArg(name, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *BaseChain) children() []Link {
	return c.links
}

func (c *BaseChain) Permissions() []cfg.Permission {
	permissions := []cfg.Permission{}
	seen := make(map[string]bool)

	links := []Link{c.Base}
	links = append(links, c.children()...)

	for _, link := range links {
		for _, perm := range link.Permissions() {
			if !seen[perm.String()] {
				seen[perm.String()] = true
				permissions = append(permissions, perm)
			}
		}
	}

	return permissions
}

func (c *BaseChain) PermissionsMap() map[cfg.Platform][]string {
	permissions := c.Permissions()
	permissionsMap := make(map[cfg.Platform][]string)

	for _, p := range permissions {
		permissionsMap[p.Platform] = append(permissionsMap[p.Platform], p.Permission)
	}

	return permissionsMap
}

func (c *BaseChain) Outputters() []Outputter {
	return c.outputters
}
