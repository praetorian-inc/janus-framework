package chain_test

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/output"
	"github.com/praetorian-inc/janus-framework/pkg/testutils/mocks/basics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiChain(t *testing.T) {
	multi := chain.NewMulti(
		chain.NewChain(basics.NewIntLink(cfg.WithArg("intOp", func(i int) int { return i + 1 }))),
		chain.NewChain(basics.NewIntLink(cfg.WithArg("intOp", func(i int) int { return i - 1 }))),
	)

	multi.Send(1)
	multi.Close()

	received := []int{}
	for output, ok := chain.RecvAs[int](multi); ok; output, ok = chain.RecvAs[int](multi) {
		received = append(received, output)
	}

	assert.Contains(t, received, 2)
	assert.Contains(t, received, 0)
}

func TestMultiChain_SendVariadic(t *testing.T) {
	add := func(i int) int { return i + 1 }
	sub := func(i int) int { return i - 1 }

	m := chain.NewMulti(
		chain.NewChain(basics.NewStrLink(), basics.NewStrIntLink(), basics.NewIntLink(cfg.WithArg("intOp", add))),
		chain.NewChain(basics.NewStrLink(), basics.NewStrIntLink(), basics.NewIntLink(cfg.WithArg("intOp", sub))),
	)

	m.Send("10", "20", "30")
	m.Close()

	received := []int{}
	for output, ok := chain.RecvAs[int](m); ok; output, ok = chain.RecvAs[int](m) {
		received = append(received, output)
	}

	expected := []int{9, 11, 19, 21, 29, 31}

	assert.Len(t, received, len(expected))
	for _, e := range expected {
		assert.Contains(t, received, e)
	}
}

func TestMultiChain_Outputters(t *testing.T) {
	multi := chain.NewMulti(
		chain.NewChain(basics.NewIntLink(cfg.WithArg("intOp", func(i int) int { return i + 1 }))),
		chain.NewChain(basics.NewIntLink(cfg.WithArg("intOp", func(i int) int { return i - 1 }))),
	).WithConfigs(
		cfg.WithArg("jsonoutfile", "test.json"),
	).WithOutputters(
		output.NewJSONOutputter(),
	)

	multi.Send(1)
	multi.Close()
	multi.Wait()

	content, err := os.ReadFile("test.json")
	assert.NoError(t, err)
	assert.True(t, string(content) == "[0,2]\n" || string(content) == "[2,0]\n") // order is nondeterministic

	os.Remove("test.json")
}

func TestMultiChain_OutputtersBeforeConfigs(t *testing.T) {
	multi := chain.NewMulti(
		chain.NewChain(basics.NewIntLink(cfg.WithArg("intOp", func(i int) int { return i + 1 }))),
		chain.NewChain(basics.NewIntLink(cfg.WithArg("intOp", func(i int) int { return i - 1 }))),
	).WithOutputters(
		output.NewJSONOutputter(),
	).WithConfigs(
		cfg.WithArg("jsonoutfile", "test.json"),
	)

	multi.Send(1)
	multi.Close()
	multi.Wait()

	content, err := os.ReadFile("test.json")
	assert.NoError(t, err)
	assert.True(t, string(content) == "[0,2]\n" || string(content) == "[2,0]\n") // order is nondeterministic

	os.Remove("test.json")
}

func TestMultichain_ArgsRequired(t *testing.T) {
	multi := chain.NewMulti(
		chain.NewChain(basics.NewParamsLink()),
		chain.NewChain(basics.NewParamsLink()),
	)

	multi.Send("data")
	multi.Close()
	multi.Wait()

	assert.ErrorContains(t, multi.Error(), "parameter \"required\" is required")
}

func TestMultichain_ArgsRegex(t *testing.T) {
	m := chain.NewMulti(
		chain.NewChain(basics.NewRegexChecker(regexp.MustCompile("^[0-9]+$"))),
		chain.NewChain(basics.NewRegexChecker(regexp.MustCompile("^[0-9]+$"))),
	).WithConfigs(
		cfg.WithArg("argument", "does not match"),
	)

	m.Send("123")
	m.Close()
	m.Wait()

	assert.ErrorContains(t, m.Error(), "error validating regex: value \"does not match\" does not match regex \"^[0-9]+$\"")
}

func TestMultichain_ArgsInvalid(t *testing.T) {
	m := chain.NewMulti(
		chain.NewChain(basics.NewParamsLink()),
		chain.NewChain(basics.NewParamsLink()),
	).WithConfigs(
		cfg.WithArg("optional", 123), // should be string
		cfg.WithArg("required", ""),
	)

	m.Send("123")
	m.Close()
	m.Wait()

	assert.ErrorContains(t, m.Error(), "parameter \"optional\" expects type \"string\", but argument value is type \"int\"")
}

func TestMultichain_ArgsFromList(t *testing.T) {
	didRun := false
	assertFunc := func(link chain.Link) {
		didRun = true

		arg1 := link.Arg("string")
		arg2 := link.Arg("stringSlice")
		arg3 := link.Arg("int")
		arg4 := link.Arg("writer")
		arg5 := link.Arg("stringWithDefault")
		arg6 := link.Arg("anotherSlice")

		stringArg, err := cfg.As[string](arg1)
		assert.NoError(t, err)
		assert.Equal(t, "hello", stringArg)

		stringSliceArg, err := cfg.As[[]string](arg2)
		assert.NoError(t, err)
		assert.Equal(t, []string{"hello", "world"}, stringSliceArg)

		intArg, err := cfg.As[int](arg3)
		assert.NoError(t, err)
		assert.Equal(t, 123, intArg)

		writerArg, err := cfg.As[io.Writer](arg4)
		assert.NoError(t, err)
		assert.NotNil(t, writerArg)

		stringWithDefaultArg, err := cfg.As[string](arg5)
		assert.NoError(t, err)
		assert.Equal(t, "default value", stringWithDefaultArg)

		writerArg.Write([]byte("hello"))

		anotherSliceArg, err := cfg.As[[]string](arg6)
		assert.NoError(t, err)
		assert.Equal(t, []string{}, anotherSliceArg)
	}

	m := chain.NewMulti(
		chain.NewChain(basics.NewCLIArgsLink(assertFunc)),
		chain.NewChain(basics.NewCLIArgsLink(assertFunc)),
	).WithConfigs(
		cfg.WithCLIArgs([]string{
			"-s", "hello",
			"-slice", "hello,world",
			"-i", "123",
			"-w", "test.json",
			"-anotherslice", "",
		}),
	)

	m.Send("")
	m.Close()
	m.Wait()

	assert.True(t, didRun, "link did not run")
	assert.NoError(t, m.Error())

	lines, err := os.ReadFile("test.json")
	assert.NoError(t, err)
	assert.Equal(t, "hellohello", string(lines))

	os.Remove("test.json")
}

func TestMultichain_Strictness_Lax(t *testing.T) {
	c := chain.NewMulti(
		chain.NewChain(basics.NewStrLink(), basics.NewProcessErrorLink()),
		chain.NewChain(basics.NewStrLink(), basics.NewProcessErrorLink()),
	).WithStrictness(chain.Lax)

	c.Send(1)   // trigger ConversionError
	c.Send("1") // ProcessErrorLink will create a ProcessError
	c.Close()
	c.Wait()
	err := c.Error()

	assert.NoError(t, err, "did not expect error from Lax chain")
}

func TestMultichain_Strictness_Moderate(t *testing.T) {
	c := chain.NewMulti(
		chain.NewChain(basics.NewStrLink(), basics.NewProcessErrorLink()),
		chain.NewChain(basics.NewStrLink(), basics.NewProcessErrorLink()),
	)

	c.Send(1)   // trigger ConversionError
	c.Send("1") // ProcessErrorLink will create a ProcessError
	c.Close()
	c.Wait()
	err := c.Error()

	assert.Error(t, err, "expected error from Moderate chain")
}

func TestMultichain_Strictness_Strict(t *testing.T) {
	c := chain.NewMulti(
		chain.NewChain(basics.NewStrLink(), basics.NewProcessErrorLink()),
		chain.NewChain(basics.NewStrLink(), basics.NewProcessErrorLink()),
	).WithStrictness(chain.Strict)

	c.Send(1)   // trigger ConversionError
	c.Send("1") // ProcessErrorLink will create a ProcessError
	c.Close()
	c.Wait()
	err := c.Error()

	assert.Error(t, err, "expected error from Strict chain")
}

func TestMultichain_ErrorConditions_Initialize(t *testing.T) {
	c := chain.NewMulti(
		chain.NewChain(
			basics.NewErrorLink(cfg.WithArg("errorAt", "initialize")),
		),
		chain.NewChain(
			basics.NewErrorLink(cfg.WithArg("errorAt", "initialize")),
		),
	)

	c.Send("123")
	c.Close()
	c.Wait()
	assert.ErrorContains(t, c.Error(), "initialize error")
}

func TestMultichain_ErrorConditions_Process(t *testing.T) {
	c := chain.NewMulti(
		chain.NewChain(
			basics.NewErrorLink(cfg.WithArg("errorAt", "process")),
		),
		chain.NewChain(
			basics.NewErrorLink(cfg.WithArg("errorAt", "process")),
		),
	)

	c.Send("123")
	c.Close()
	c.Wait()
	assert.ErrorContains(t, c.Error(), "process error:")
}

func TestMultichain_ErrorConditions_Complete(t *testing.T) {
	c := chain.NewMulti(
		chain.NewChain(
			basics.NewErrorLink(cfg.WithArg("errorAt", "complete")),
		),
		chain.NewChain(
			basics.NewErrorLink(cfg.WithArg("errorAt", "complete")),
		),
	)

	c.Send("123")
	c.Close()
	c.Wait()
	assert.ErrorContains(t, c.Error(), "complete error")
}

func TestMultiChain_WithAddedLinks(t *testing.T) {
	c := chain.NewMulti(
		chain.NewChain(basics.NewStrLink(), basics.NewStrIntLink()),
		chain.NewChain(basics.NewStrLink(), basics.NewStrIntLink()),
	).WithAddedLinks(
		basics.NewIntLink(cfg.WithArg("intOp", func(i int) int { return i + 1 })),
	)

	c.Send("123")
	c.Close()
	c.Wait()

	assert.EqualError(t, c.Error(), "WithAddedLinks is not supported on MultiChain")
}

func TestMultiChain_Logging(t *testing.T) {
	w := &bytes.Buffer{}

	m := chain.NewMulti(
		chain.NewChain(basics.NewLoggingLink()).WithLogWriter(w).WithName("chain1"),
		chain.NewChain(basics.NewLoggingLink()).WithLogWriter(w).WithName("chain2"),
	)

	m.WithName("multi-chain")
	m.WithLogColoring(false)

	m.Send(basics.Msg{Level: slog.LevelInfo, Message: "test"})
	m.Close()
	m.Wait()

	assert.Contains(t, w.String(), "level=INFO link=multi-chain/chain1/*basics.LoggingLink msg=test")
	assert.Contains(t, w.String(), "level=INFO link=multi-chain/chain2/*basics.LoggingLink msg=test")
}

func TestMultiChain_Logging_WithLogLevel(t *testing.T) {
	w := &bytes.Buffer{}
	c := chain.NewMulti(
		chain.NewChain(basics.NewLoggingLink()).WithLogWriter(w).WithName("chain1"),
		chain.NewChain(basics.NewLoggingLink()).WithLogWriter(w).WithName("chain2"),
	)

	c.WithName("multi-chain")
	c.WithLogColoring(false)
	c.WithLogLevel(slog.LevelWarn)

	c.Send(basics.Msg{Level: slog.LevelDebug, Message: "Debug message"})
	c.Send(basics.Msg{Level: slog.LevelInfo, Message: "Info message"})
	c.Send(basics.Msg{Level: slog.LevelWarn, Message: "Warn message"})
	c.Send(basics.Msg{Level: slog.LevelError, Message: "Error message"})
	c.Close()
	c.Wait()

	formatted := strings.ReplaceAll(w.String(), "\\n", "\n")

	require.Contains(t, w.String(), "level=WARN link=multi-chain/chain1/*basics.LoggingLink msg=\"Warn message\"", formatted)
	require.Contains(t, w.String(), "level=ERROR link=multi-chain/chain1/*basics.LoggingLink msg=\"Error message\"", formatted)
	require.NotContains(t, w.String(), "level=DEBUG link=multi-chain/chain1/*basics.LoggingLink msg=\"Debug message\"", formatted)
	require.NotContains(t, w.String(), "level=INFO link=multi-chain/chain1/*basics.LoggingLink msg=\"Info message\"", formatted)

	require.Contains(t, w.String(), "level=WARN link=multi-chain/chain2/*basics.LoggingLink msg=\"Warn message\"", formatted)
	require.Contains(t, w.String(), "level=ERROR link=multi-chain/chain2/*basics.LoggingLink msg=\"Error message\"", formatted)
	require.NotContains(t, w.String(), "level=DEBUG link=multi-chain/chain2/*basics.LoggingLink msg=\"Debug message\"", formatted)
	require.NotContains(t, w.String(), "level=INFO link=multi-chain/chain2/*basics.LoggingLink msg=\"Info message\"", formatted)
}
