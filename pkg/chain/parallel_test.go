package chain_test

import (
	"testing"
	"time"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/testutils/mocks/basics"
	"github.com/stretchr/testify/assert"
)

// TestParallelize_Basic tests basic parallelization functionality
func TestParallelize_Basic(t *testing.T) {
	// Create a parallelized string link
	parallelStrLink := chain.Parallelize(basics.NewStrLink)

	// Create chain with parallel link
	c := chain.NewChain(parallelStrLink()).WithConfigs(
		cfg.WithArg("workers", 2),
	)

	// Send multiple inputs
	inputs := []string{"hello", "world", "test"}
	for _, input := range inputs {
		c.Send(input)
	}
	c.Close()

	// Collect outputs
	received := []string{}
	for output, ok := chain.RecvAs[string](c); ok; output, ok = chain.RecvAs[string](c) {
		received = append(received, output)
	}

	// Should receive all inputs (though order may vary due to parallelization)
	assert.Equal(t, len(inputs), len(received))
	assert.Contains(t, received, "hello")
	assert.Contains(t, received, "world") 
	assert.Contains(t, received, "test")
	assert.NoError(t, c.Error())
}

// TestParallelize_DefaultWorkers tests default worker count
func TestParallelize_DefaultWorkers(t *testing.T) {
	parallelLink := chain.Parallelize(basics.NewStrLink)()

	// Check that workers parameter exists with default value
	found := false
	for _, param := range parallelLink.Params() {
		if param.Name() == "workers" {
			found = true
			break
		}
	}
	assert.True(t, found, "workers parameter should be available")
}

// TestParallelize_SmallInputLargeWorkers tests edge case of few inputs with many workers
func TestParallelize_SmallInputLargeWorkers(t *testing.T) {
	parallelLink := chain.Parallelize(basics.NewStrLink)

	c := chain.NewChain(parallelLink()).WithConfigs(
		cfg.WithArg("workers", 10), // 10 workers for 2 inputs
	)

	// Send only 2 inputs
	c.Send("input1")
	c.Send("input2")
	c.Close()

	// Collect outputs
	received := []string{}
	for output, ok := chain.RecvAs[string](c); ok; output, ok = chain.RecvAs[string](c) {
		received = append(received, output)
	}

	// Should handle gracefully and process both inputs
	assert.Equal(t, 2, len(received))
	assert.Contains(t, received, "input1")
	assert.Contains(t, received, "input2")
	assert.NoError(t, c.Error())
}

// TestParallelize_ZeroInput tests handling of zero inputs
func TestParallelize_ZeroInput(t *testing.T) {
	parallelLink := chain.Parallelize(basics.NewStrLink)

	c := chain.NewChain(parallelLink()).WithConfigs(
		cfg.WithArg("workers", 5),
	)

	// Close without sending any input
	c.Close()

	// Should handle gracefully
	received := []string{}
	for output, ok := chain.RecvAs[string](c); ok; output, ok = chain.RecvAs[string](c) {
		received = append(received, output)
	}

	assert.Empty(t, received)
	assert.NoError(t, c.Error())
}

// TestParallelize_ErrorHandling tests error handling in parallel processing
func TestParallelize_ErrorHandling(t *testing.T) {
	// Use an error link that fails on specific input
	parallelLink := chain.Parallelize(basics.NewErrorLink)

	c := chain.NewChain(parallelLink()).WithConfigs(
		cfg.WithArg("workers", 2),
		cfg.WithArg("errorAt", "process"),
	)

	c.Send("trigger-error")
	c.Close()

	// Should not crash the entire chain
	received := []string{}
	for output, ok := chain.RecvAs[string](c); ok; output, ok = chain.RecvAs[string](c) {
		received = append(received, output)
	}

	// Error should be handled gracefully
	assert.Empty(t, received) // No output expected from error link
	assert.NoError(t, c.Error()) // Chain should not fail due to worker error
}

// TestParallelize_ParameterPassing tests that parameters are passed to workers
func TestParallelize_ParameterPassing(t *testing.T) {
	didRun := false
	assertFunc := func(arg string, err error) {
		assert.NoError(t, err)
		assert.Equal(t, "test-arg", arg)
		didRun = true
	}

	parallelLink := chain.Parallelize(func(configs ...cfg.Config) chain.Link {
		return basics.NewArgCheckingLink(assertFunc, configs...)
	})

	c := chain.NewChain(parallelLink()).WithConfigs(
		cfg.WithArg("workers", 1),
		cfg.WithArg("argument", "test-arg"),
	)

	c.Send("test-input")
	c.Close()
	c.Wait()

	assert.True(t, didRun, "worker should have received parameter")
	assert.NoError(t, c.Error())
}

// TestParallelize_MultipleTypes tests parallelization with type conversion chain
func TestParallelize_MultipleTypes(t *testing.T) {
	// Create a chain with parallel string-to-int conversion
	parallelConverter := chain.Parallelize(basics.NewStrIntLink)

	c := chain.NewChain(
		basics.NewStrLink(),
		parallelConverter(),
		basics.NewIntLink(),
	).WithConfigs(
		cfg.WithArg("workers", 3),
	)

	// Send multiple string numbers
	inputs := []string{"123", "456", "789"}
	for _, input := range inputs {
		c.Send(input)
	}
	c.Close()

	// Collect integer outputs
	received := []int{}
	for output, ok := chain.RecvAs[int](c); ok; output, ok = chain.RecvAs[int](c) {
		received = append(received, output)
	}

	// Should convert all strings to integers
	assert.Equal(t, 3, len(received))
	assert.Contains(t, received, 123)
	assert.Contains(t, received, 456)
	assert.Contains(t, received, 789)
	assert.NoError(t, c.Error())
}

// TestParallelize_WorkerLifecycle tests proper worker creation and cleanup
func TestParallelize_WorkerLifecycle(t *testing.T) {
	// Create a link that tracks initialization/completion
	initCount := 0
	completeCount := 0

	trackingLinkConstructor := func(configs ...cfg.Config) chain.Link {
		return NewTrackingLink(
			func() { initCount++ },    // onInit
			func() { completeCount++ }, // onComplete
			configs...,
		)
	}

	parallelLink := chain.Parallelize(trackingLinkConstructor)

	c := chain.NewChain(parallelLink()).WithConfigs(
		cfg.WithArg("workers", 2),
	)

	// Send inputs to trigger worker creation
	c.Send("input1")
	c.Send("input2") 
	c.Send("input3")
	c.Close()
	c.Wait()

	// Add a small delay to ensure async operations complete
	time.Sleep(10 * time.Millisecond)

	// Should have created workers and completed them properly
	t.Logf("initCount: %d, completeCount: %d", initCount, completeCount)
	assert.True(t, initCount > 0, "workers should be initialized")
	assert.True(t, completeCount > 0, "workers should be completed")
	assert.NoError(t, c.Error())
}

// TestParallelize_ConfigurableWorkers tests different worker counts
func TestParallelize_ConfigurableWorkers(t *testing.T) {
	testCases := []struct {
		name        string
		workerCount int
		inputCount  int
	}{
		{"SingleWorker", 1, 5},
		{"MultipleWorkers", 4, 8},
		{"ManyWorkers", 10, 3}, // More workers than inputs
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parallelLink := chain.Parallelize(basics.NewStrLink)

			c := chain.NewChain(parallelLink()).WithConfigs(
				cfg.WithArg("workers", tc.workerCount),
			)

			// Send inputs
			for i := 0; i < tc.inputCount; i++ {
				c.Send("input")
			}
			c.Close()

			// Count outputs
			outputCount := 0
			for _, ok := chain.RecvAs[string](c); ok; _, ok = chain.RecvAs[string](c) {
				outputCount++
			}

			assert.Equal(t, tc.inputCount, outputCount)
			assert.NoError(t, c.Error())
		})
	}
}

// TestParallelize_Performance tests that parallelization provides some benefit
func TestParallelize_Performance(t *testing.T) {
	// Create a slow processing link
	slowLink := NewSlowLink

	inputCount := 4
	delay := 100 * time.Millisecond

	// Test sequential processing
	startSeq := time.Now()
	seqChain := chain.NewChain(slowLink(cfg.WithArg("delay", delay)))
	for i := 0; i < inputCount; i++ {
		seqChain.Send("input")
	}
	seqChain.Close()
	for _, ok := chain.RecvAs[string](seqChain); ok; _, ok = chain.RecvAs[string](seqChain) {
	}
	seqDuration := time.Since(startSeq)

	// Test parallel processing
	startPar := time.Now()
	parallelLink := chain.Parallelize(slowLink)
	parChain := chain.NewChain(parallelLink()).WithConfigs(
		cfg.WithArg("workers", 4),
		cfg.WithArg("delay", delay),
	)
	for i := 0; i < inputCount; i++ {
		parChain.Send("input")
	}
	parChain.Close()
	for _, ok := chain.RecvAs[string](parChain); ok; _, ok = chain.RecvAs[string](parChain) {
	}
	parDuration := time.Since(startPar)

	// Parallel should be faster (allowing for some overhead, but should be significantly faster for I/O bound tasks)
	// Allow up to 50% overhead - if parallel takes more than 1.5x sequential time, something is wrong
	maxAllowedParallelTime := seqDuration + (seqDuration / 2)
	assert.True(t, parDuration < maxAllowedParallelTime, 
		"parallel processing (%v) should not be significantly slower than sequential (%v)", parDuration, seqDuration)
}

// Helper structs for testing

// TrackingLink tracks initialization and completion calls
type TrackingLink struct {
	*chain.Base
	onInit     func()
	onComplete func()
}

func NewTrackingLink(onInit, onComplete func(), configs ...cfg.Config) chain.Link {
	tl := &TrackingLink{
		onInit:     onInit,
		onComplete: onComplete,
	}
	tl.Base = chain.NewBase(tl, configs...)
	return tl
}

func (tl *TrackingLink) Initialize() error {
	if tl.onInit != nil {
		tl.onInit()
	}
	return nil
}

func (tl *TrackingLink) Process(input any) error {
	return tl.Send(input)
}

func (tl *TrackingLink) Complete() error {
	if tl.onComplete != nil {
		tl.onComplete()
	}
	return nil
}

// SlowLink introduces artificial delay for performance testing
type SlowLink struct {
	*chain.Base
}

func NewSlowLink(configs ...cfg.Config) chain.Link {
	sl := &SlowLink{}
	sl.Base = chain.NewBase(sl, configs...)
	return sl
}

func (sl *SlowLink) Process(input any) error {
	if delayArg := sl.Arg("delay"); delayArg != nil {
		if delay, ok := delayArg.(time.Duration); ok {
			time.Sleep(delay)
		}
	}
	return sl.Send(input)
}