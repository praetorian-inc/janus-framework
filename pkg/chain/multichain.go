package chain

import (
	"fmt"

	"github.com/praetorian-inc/janus-framework/pkg/util"
)

type MultiChain struct {
	*BaseChain
	chanIns []chan any
}

func NewMulti(chains ...Link) Chain {
	m := &MultiChain{
		BaseChain: &BaseChain{
			links:  chains,
			chanIn: make(chan any),
		},
		chanIns: createChannels(chains...),
	}
	m.BaseChain.Base = NewBase(m)
	m.super = m

	for _, chain := range chains {
		if chain.isClaimed() {
			m.handleError(fmt.Errorf("link %s is in-use by another chain", chain.Name()))
		}
		chain.claim()
		if chain.Error() != nil {
			m.handleError(chain.Error())
		}
		chain.AddAncestor(&m.Base.name)
	}

	return m
}

func (m *MultiChain) WithAddedLinks(_ ...Link) Chain {
	m.handleError(fmt.Errorf("WithAddedLinks is not supported on MultiChain"))
	return m.super
}

func createChannels(chains ...Link) []chan any {
	chanIns := make([]chan any, len(chains))
	for i := range chains {
		chanIns[i] = make(chan any)
	}
	return chanIns
}

func (m *MultiChain) Send(values ...any) error {
	m.startIfUnstarted()
	m.errLock.Lock()
	defer m.errLock.Unlock()

	if err := m.getError(); err != nil {
		return fmt.Errorf("chain is in error state: %w", err)
	}

	for _, v := range values {
		m.chanIn <- v
	}
	return nil
}

func (m *MultiChain) disperseInput(input any) {
	for _, chanIn := range m.chanIns {
		chanIn <- input
	}
}

func (m *MultiChain) Process(v any) error {
	for _, chanIn := range m.chanIns {
		chanIn <- v
	}
	return nil
}

func (m *MultiChain) Close() {
	m.startIfUnstarted()
	m.closeChanInOnce()
}

func (m *MultiChain) Wait() {
	m.startIfUnstarted()

	m.errLock.Lock()
	err := m.getError()
	m.errLock.Unlock()

	if err != nil {
		return
	}

	if len(m.outputters) == 0 {
		// caller has called Wait() with no outputters.
		// c.wgOut.Wait() will deadlock if we do not empty this channel.
		util.EmptyChannel(m.channel())
	}

	m.wgOut.Wait()
}

func (m *MultiChain) startIfUnstarted() {
	if !m.hasStarted() {
		m.start(m.chanIn, m.handleError, m.strictness)
	}
	m.setStarted()
}

func (m *MultiChain) closeChanInOnce() {
	m.closeChanIn.Do(func() {
		close(m.chanIn)
	})
}

func (m *MultiChain) Error() error {
	return m.BaseChain.Error()
}

func (m *MultiChain) start(prevChan chan any, errHandler func(error), strictness Strictness) {
	m.initializeLogger()

	if err := m.resetParams(); err != nil {
		errHandler(err)
		return
	}

	for _, outputter := range m.outputters {
		if err := m.startOutputter(outputter); err != nil {
			errHandler(err)
		}
	}

	go m.startDisperser(prevChan)

	for i, child := range m.children() {
		m.startChild(child, m.chanIns[i], errHandler, strictness)
	}

	m.wgOut.Add(1)
	go m.collectOutput()
}

func (m *MultiChain) startOutputter(outputter Outputter) error {
	err := m.setArgs(outputter)
	if err != nil {
		return err
	}

	err = outputter.Initialize()
	if err != nil {
		return err
	}

	return nil
}

func (m *MultiChain) startDisperser(prevChan chan any) {
	defer func() {
		for _, ch := range m.chanIns {
			close(ch)
		}
	}()

	for input := range prevChan {
		m.disperseInput(input)
	}
}

func (m *MultiChain) startChild(child Link, prevChan chan any, errHandler func(error), strictness Strictness) (chan any, error) {
	if err := m.setArgs(child); err != nil {
		errHandler(err)
		return nil, err
	}

	go child.start(prevChan, errHandler, strictness)
	return child.channel(), nil
}

func (m *MultiChain) collectOutput() {
	defer func() {
		m.flushOutputItems()
		close(m.channel())
		m.closeOutputters()
		m.wgOut.Done()
	}()

	for _, child := range m.children() {
		for v := range child.channel() {
			m.output(v)
		}
	}
}
