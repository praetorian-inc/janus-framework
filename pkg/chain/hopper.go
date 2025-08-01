package chain

type Hopper struct {
	*Base
	chains []Chain
}

func NewHopper(chains ...Chain) Link {
	h := &Hopper{chains: chains}
	h.Base = NewBase(h)
	return h
}

func (h *Hopper) start(_ chan any, _ func(error), _ Strictness) {
	go h.processLoop()
}

func (h *Hopper) processLoop() {
	defer close(h.channel())
	for _, chain := range h.chains {
		h.collectOutput(chain)
	}
}

func (h *Hopper) StartForUnitTests() {
	go h.start(nil, nil, h.strictness)
}

func (h *Hopper) collectOutput(chain Chain) {
	for output := range chain.channel() {
		h.Send(output)
	}
}
