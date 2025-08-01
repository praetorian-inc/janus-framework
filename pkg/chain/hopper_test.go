package chain_test

import (
	"testing"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/testutils/mocks/basics"
	"github.com/stretchr/testify/assert"
)

func TestHopper(t *testing.T) {
	chain1 := chain.NewChain(
		basics.NewInterfaceLink(),
		basics.NewStrLink(),
	)

	chain2 := chain.NewChain(
		basics.NewStrLink(),
	)

	hopper := chain.NewHopper(chain1, chain2)

	go func() {
		chain1.Send(&basics.Mockable{MockMsg: "hello"})
		chain2.Send("world")
		chain1.Close()
		chain2.Close()
	}()

	hopper.(*chain.Hopper).StartForUnitTests()

	received := []string{}
	for output, ok := chain.RecvAs[string](hopper); ok; output, ok = chain.RecvAs[string](hopper) {
		received = append(received, output)
	}

	assert.Contains(t, received, "hello")
	assert.Contains(t, received, "world")
}
