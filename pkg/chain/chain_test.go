package chain_test

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/output"
	"github.com/praetorian-inc/janus-framework/pkg/testutils"
	"github.com/praetorian-inc/janus-framework/pkg/testutils/mocks"
	"github.com/praetorian-inc/janus-framework/pkg/testutils/mocks/basics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChain_SingleLink(t *testing.T) {
	c := chain.NewChain(
		basics.NewStrLink(),
	)

	c.Send("hello")
	c.Send("hello")
	c.Send("hello")
	c.Close()

	received := []string{}
	for output, ok := chain.RecvAs[string](c); ok; output, ok = chain.RecvAs[string](c) {
		received = append(received, output)
	}

	assert.Equal(t, []string{"hello", "hello", "hello"}, received)
	assert.NoError(t, c.Error())
}

func TestChain_Simple(t *testing.T) {
	c := chain.NewChain(
		basics.NewStrLink(),
		basics.NewStrIntLink(),
		basics.NewIntLink(),
	)

	c.Send("123")
	c.Close()

	received := []int{}
	for output, ok := chain.RecvAs[int](c); ok; output, ok = chain.RecvAs[int](c) {
		received = append(received, output)
	}

	assert.Equal(t, []int{123}, received)
	assert.NoError(t, c.Error())
}

func TestChain_NoInput(t *testing.T) {
	c := chain.NewChain(
		basics.NewStrLink(),
	)

	assert.NotPanics(t, func() {
		c.Close()
		c.Wait()
	})
	assert.NoError(t, c.Error())
}

func TestChain_SendVariadic(t *testing.T) {
	c := chain.NewChain(
		basics.NewStrLink(),
		basics.NewStrIntLink(),
		basics.NewIntLink(),
	)

	c.Send("123", "456", "789")
	c.Close()

	received := []int{}
	for output, ok := chain.RecvAs[int](c); ok; output, ok = chain.RecvAs[int](c) {
		received = append(received, output)
	}

	assert.Equal(t, []int{123, 456, 789}, received)
	assert.NoError(t, c.Error())
}

func TestChain_StructConversion(t *testing.T) {
	chain := chain.NewChain(
		basics.NewStrStructLink(),
		basics.NewStrLink(),
	)

	type moreThanStrStruct struct {
		Str        string
		Additional string
	}

	chain.Send(moreThanStrStruct{"123", ""})
	chain.Close()
}

func TestChain_Interface(t *testing.T) {
	c := chain.NewChain(
		basics.NewInterfaceLink(),
		basics.NewStrLink(),
	)

	c.Send(&basics.Mockable{MockMsg: "mocking"})
	c.Close()

	received, ok := chain.RecvAs[string](c)
	assert.True(t, ok)
	assert.Equal(t, "mocking", received)
	assert.NoError(t, c.Error())
}

func TestChain_NilInput(t *testing.T) {
	w := &bytes.Buffer{}
	c := chain.NewChain(
		basics.NewInterfaceLink(),
		basics.NewStrLink(),
	)
	c.WithLogWriter(w)
	c.WithLogLevel(slog.LevelDebug)

	c.Send(nil)
	c.Close()

	received, ok := chain.RecvAs[string](c)
	assert.False(t, ok)
	assert.Empty(t, received)
	assert.NoError(t, c.Error())

	assert.Contains(t, w.String(), "input is invalid")
}

func TestChain_ChainOfChain(t *testing.T) {
	c := chain.NewChain(
		chain.NewChain(
			basics.NewStrLink(),
			basics.NewStrIntLink(),
		),
		basics.NewIntLink(),
	)

	c.Send("123")
	c.Close()

	received := []int{}
	for output, ok := chain.RecvAs[int](c); ok; output, ok = chain.RecvAs[int](c) {
		received = append(received, output)
	}

	assert.Equal(t, []int{123}, received)
	assert.NoError(t, c.Error())
}

func TestChain_Complete(t *testing.T) {
	c := chain.NewChain(
		basics.NewStrLink(),
		mocks.NewCompleterLink(),
	)

	c.Send("123")
	c.Close()

	received, ok := chain.RecvAs[string](c)
	assert.True(t, ok)
	assert.Equal(t, "completed", received)
	assert.NoError(t, c.Error())
}

func TestChain_Params(t *testing.T) {
	chain := chain.NewChain(
		basics.NewParamsLink(),
		basics.NewStrLink(),
	).WithOutputters(
		output.NewWriterOutputter(),
	)

	assert.Equal(t, 5, len(chain.Params())) // 3 from paramsLink, 1 from strLink, 1 from outputter
	assert.True(t, chain.HasParam("optional"))
	assert.True(t, chain.HasParam("required"))
	assert.True(t, chain.HasParam("default"))
	assert.True(t, chain.HasParam("writer"))
	assert.False(t, chain.HasParam("not_a_param"))
	assert.NoError(t, chain.Error())
}

func TestChain_Args(t *testing.T) {
	didRun := false
	assertFunc := func(arg string, err error) {
		assert.NoError(t, err)
		assert.Equal(t, "123", arg)
		didRun = true
	}

	chain := chain.NewChain(
		basics.NewArgCheckingLink(assertFunc),
	).WithConfigs(
		cfg.WithArg("argument", "123"),
	)

	chain.Send("123")
	chain.Close()
	chain.Wait()

	assert.True(t, didRun, "link did not run")
	assert.NoError(t, chain.Error())
}

func TestChain_LinkArgOverride(t *testing.T) {
	didRun := false
	assertFunc := func(arg string, err error) {
		assert.NoError(t, err)
		assert.Equal(t, "123", arg, "links arguments should override chain arguments")
		didRun = true
	}

	chain := chain.NewChain(
		basics.NewArgCheckingLink(assertFunc, cfg.WithArg("argument", "123")),
	).WithConfigs(
		cfg.WithArg("argument", "456"),
	)

	chain.Send("123")
	chain.Close()
	chain.Wait()

	assert.True(t, didRun, "link did not run")
	assert.NoError(t, chain.Error())
}

func TestChain_ArgsDefault(t *testing.T) {
	didRun := false
	assertFunc := func(arg string, err error) {
		assert.NoError(t, err)
		assert.Equal(t, "default value", arg)
		didRun = true
	}

	chain := chain.NewChain(
		basics.NewArgCheckingLink(assertFunc),
	)

	chain.Send("123")
	chain.Close()
	chain.Wait()

	assert.True(t, didRun, "link did not run")
	assert.NoError(t, chain.Error())
}

func TestChain_ArgsRequired(t *testing.T) {
	c := chain.NewChain(
		basics.NewParamsLink(), // requires "required" argument
	)

	c.Send("123")
	c.Close()
	c.Wait()

	assert.ErrorContains(t, c.Error(), "parameter \"required\" is required")
}

func TestChain_ArgsRegex(t *testing.T) {
	c := chain.NewChain(
		basics.NewRegexChecker(regexp.MustCompile("^[0-9]+$")),
	).WithConfigs(
		cfg.WithArg("argument", "does not match"),
	)

	c.Send("123")
	c.Close()
	c.Wait()

	assert.ErrorContains(t, c.Error(), "error validating regex: value \"does not match\" does not match regex \"^[0-9]+$\"")
}

func TestChain_ArgsInvalid(t *testing.T) {
	c := chain.NewChain(
		basics.NewParamsLink(),
	).WithConfigs(
		cfg.WithArg("optional", 123), // should be string
		cfg.WithArg("required", ""),
	)

	c.Send("123")
	c.Close()
	c.Wait()

	assert.ErrorContains(t, c.Error(), "parameter \"optional\" expects type \"string\", but argument value is type \"int\"")
}

func TestChain_ArgsFromList(t *testing.T) {
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

	c := chain.NewChain(
		basics.NewCLIArgsLink(assertFunc),
	).WithConfigs(
		cfg.WithCLIArgs([]string{
			"-s", "hello",
			"-slice", "hello,world",
			"-i", "123",
			"-w", "test.json",
			"-anotherslice", "",
		}),
	)

	c.Send("")
	c.Close()
	c.Wait()

	assert.True(t, didRun, "link did not run")
	assert.NoError(t, c.Error())

	lines, err := os.ReadFile("test.json")
	assert.NoError(t, err)
	assert.Equal(t, "hello", string(lines))

	os.Remove("test.json")
}

func TestChain_ArgsFromList_Regex(t *testing.T) {
	c := chain.NewChain(
		basics.NewRegexChecker(regexp.MustCompile("^[0-9]+$")),
	).WithConfigs(
		cfg.WithCLIArgs([]string{"-argument", "does not match"}),
	)

	c.Send("123")
	c.Close()
	c.Wait()

	assert.ErrorContains(t, c.Error(), "error validating regex: value \"does not match\" does not match regex \"^[0-9]+$\"")
}

func TestChain_ArgsFromList_Invalid(t *testing.T) {
	c := chain.NewChain(
		basics.NewParamsLink(),
	).WithConfigs(
		cfg.WithCLIArgs([]string{"-default", "should be integer", "-required", "present"}),
	)

	c.Send("123")
	c.Close()
	c.Wait()

	assert.ErrorContains(t, c.Error(), `failed to convert value "should be integer" to type "int": strconv.Atoi: parsing "should be integer": invalid syntax`)
}

func TestChain_Outputters(t *testing.T) {
	w := &bytes.Buffer{}
	chain := chain.NewChain(
		basics.NewStrLink(),
		basics.NewStrIntLink(),
	).WithConfigs(
		cfg.WithArg("writer", w),
		cfg.WithArg("jsonoutfile", "test.json"),
	).WithOutputters(
		output.NewWriterOutputter(),
		output.NewJSONOutputter(),
	)

	chain.Send("123")
	chain.Close()
	chain.Wait()

	require.NoError(t, chain.Error())
	assert.Equal(t, "123\n", w.String())

	content, err := os.ReadFile("test.json")
	assert.NoError(t, err)
	assert.Equal(t, "[123]\n", string(content))

	os.Remove("test.json")
}

func TestChain_OutputtersBeforeConfigs(t *testing.T) {
	w := &bytes.Buffer{}
	chain := chain.NewChain(
		basics.NewStrLink(),
		basics.NewStrIntLink(),
	).WithOutputters(
		output.NewWriterOutputter(),
		output.NewJSONOutputter(),
	).WithConfigs(
		cfg.WithArg("writer", w),
		cfg.WithArg("jsonoutfile", "test.json"),
	)

	chain.Send("123")
	chain.Close()
	chain.Wait()

	require.NoError(t, chain.Error())
	assert.Equal(t, "123\n", w.String())

	content, err := os.ReadFile("test.json")
	assert.NoError(t, err)
	assert.Equal(t, "[123]\n", string(content))

	os.Remove("test.json")
}

type TestUndeclaredParamOutputter struct {
	*chain.BaseOutputter
	paramValue *string
}

func (o *TestUndeclaredParamOutputter) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("declared-param", "a parameter this outputter declares"),
	}
}

func (o *TestUndeclaredParamOutputter) Initialize() error {
	// Access a parameter that this outputter didn't declare
	if profile := o.Arg("profile"); profile != nil {
		*o.paramValue = profile.(string)
	}
	return nil
}

func (o *TestUndeclaredParamOutputter) Output(any) error {
	return nil
}

func NewTestUndeclaredParamOutputter(paramValue *string, configs ...cfg.Config) chain.Outputter {
	o := &TestUndeclaredParamOutputter{paramValue: paramValue}
	o.BaseOutputter = chain.NewBaseOutputter(o, configs...)
	return o
}

func TestChain_OutputterReceivesUndeclaredParams(t *testing.T) {
	receivedParam := ""

	c := chain.NewChain(
		basics.NewProfileLink(cfg.WithArg("profile", "test-profile")),
	).WithConfigs(
		cfg.WithArg("declared-param", "test-value"),
	).WithOutputters(
		NewTestUndeclaredParamOutputter(&receivedParam),
	)

	c.Send("test")
	c.Close()
	c.Wait()

	require.NoError(t, c.Error())
	assert.Equal(t, "test-profile", receivedParam, "outputter should receive undeclared parameter from chain")
}

func TestChain_Hopper(t *testing.T) {
	chain1 := chain.NewChain(
		basics.NewInterfaceLink(),
		basics.NewStrLink(),
	)

	chain2 := chain.NewChain(
		basics.NewStrLink(),
	)

	c := chain.NewChain(
		chain.NewHopper(chain1, chain2),
		basics.NewStrIntLink(),
	)

	chain1.Send(&basics.Mockable{MockMsg: "123"})
	chain2.Send("456")
	chain1.Close()
	chain2.Close()

	received := []int{}
	for output, ok := chain.RecvAs[int](c); ok; output, ok = chain.RecvAs[int](c) {
		received = append(received, output)
	}

	assert.Contains(t, received, 123)
	assert.Contains(t, received, 456)
	assert.NoError(t, c.Error())
	assert.NoError(t, chain1.Error())
	assert.NoError(t, chain2.Error())
}

func TestChain_MultiChain(t *testing.T) {
	add := func(i int) int { return i + 1 }
	subtract := func(i int) int { return i - 1 }
	double := func(i int) int { return i * 2 }

	multi := chain.NewMulti(
		chain.NewChain(basics.NewIntLink(cfg.WithArg("intOp", add))),
		chain.NewChain(basics.NewIntLink(cfg.WithArg("intOp", subtract))),
	)

	c := chain.NewChain(
		basics.NewIntLink(),
		multi,
		basics.NewIntLink(cfg.WithArg("intOp", double)),
	)

	c.Send(1)
	c.Close()

	received := []int{}
	for output, ok := chain.RecvAs[int](c); ok; output, ok = chain.RecvAs[int](c) {
		received = append(received, output)
	}

	assert.Equal(t, []int{4, 0}, received)
	assert.NoError(t, c.Error())
}

func TestChain_Complex(t *testing.T) {
	addSuffix := func(suffix string) func(s string) string {
		return func(s string) string {
			return s + ":" + suffix
		}
	}

	multi1 := chain.NewMulti(
		chain.NewChain(basics.NewStrLink(cfg.WithArg("strOp", addSuffix("multichain1A")))),
		chain.NewChain(basics.NewStrLink(cfg.WithArg("strOp", addSuffix("multichain1B")))),
	)

	multi2 := chain.NewMulti(
		chain.NewChain(basics.NewStrLink(cfg.WithArg("strOp", addSuffix("multichain2A")))),
		chain.NewChain(basics.NewStrLink(cfg.WithArg("strOp", addSuffix("multichain2B")))),
	)

	hopper := chain.NewChain(
		chain.NewHopper(multi1, multi2),
		basics.NewStrLink(cfg.WithArg("strOp", addSuffix("hopper"))),
	)

	c := chain.NewChain(
		basics.NewStrLink(),
		hopper,
		basics.NewStrLink(cfg.WithArg("strOp", addSuffix("final"))),
	)

	multi1.Send("first")
	multi2.Send("second")

	multi1.Close()
	multi2.Close()

	expected := []string{
		"first:multichain1A:hopper:final",
		"first:multichain1B:hopper:final",
		"second:multichain2A:hopper:final",
		"second:multichain2B:hopper:final",
	}

	actual := []string{}
	for output, ok := chain.RecvAs[string](c); ok; output, ok = chain.RecvAs[string](c) {
		actual = append(actual, output)
	}

	assert.Len(t, actual, 4, "received unexpected number of outputs")
	assert.Contains(t, actual, expected[0])
	assert.Contains(t, actual, expected[1])
	assert.Contains(t, actual, expected[2])
	assert.Contains(t, actual, expected[3])
	assert.NoError(t, c.Error())
	assert.NoError(t, multi1.Error())
	assert.NoError(t, multi2.Error())
	assert.NoError(t, hopper.Error())
}

func TestChain_DependencyInjection(t *testing.T) {
	didExecute := false

	mockMethods := &testutils.MockMethods{ExecuteFn: func(cmd *exec.Cmd, lineparser func(string)) error {
		didExecute = true
		lineparser("injected1.praetorian.com")
		lineparser("injected2.praetorian.com")
		lineparser("injected3.praetorian.com")
		return nil
	}}

	c := chain.NewChain(
		mocks.NewExecutor(cfg.WithMethods(mockMethods)),
	)

	c.Send("praetorian.com")
	c.Close()

	expected := []string{
		"injected1.praetorian.com",
		"injected2.praetorian.com",
		"injected3.praetorian.com",
	}

	actual := []string{}
	for subdomain, ok := chain.RecvAs[string](c); ok; subdomain, ok = chain.RecvAs[string](c) {
		actual = append(actual, subdomain)
	}

	assert.Equal(t, expected, actual)
	assert.True(t, didExecute, "base did not execute")
	assert.NoError(t, c.Error())
}

func TestChain_ExecuteCmdError(t *testing.T) {
	c := chain.NewChain(
		mocks.NewExecutor(cfg.WithArg("cmd", "nonexistentcmd")),
	)

	c.Send("123")
	c.Close()
	c.Wait()
	err := c.Error()

	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), `exec: "nonexistentcmd": executable file not found in $PATH`))
}

func TestChain_ChainStrictness_Lax(t *testing.T) {
	c := chain.NewChain(basics.NewStrLink(), basics.NewProcessErrorLink()).WithStrictness(chain.Lax) // Lax, exit on ProcessError

	c.Send(1)   // trigger ConversionError
	c.Send("1") // ProcessErrorLink will create a ProcessError
	c.Close()
	c.Wait()
	err := c.Error()

	assert.NoError(t, err, "did not expect error from Lax chain")
}

func TestChain_ChainStrictness_Moderate(t *testing.T) {
	c := chain.NewChain(basics.NewStrLink(), basics.NewProcessErrorLink()) // Moderate, exit on ProcessError

	c.Send(1)   // trigger ConversionError
	c.Send("1") // ProcessErrorLink will create a ProcessError
	c.Close()
	c.Wait()
	err := c.Error()

	assert.Error(t, err, "expected error from Moderate chain")
}

func TestChain_ChainStrictness_Strict(t *testing.T) {
	c := chain.NewChain(basics.NewStrLink(), basics.NewProcessErrorLink()).WithStrictness(chain.Strict) // Strict, exit on ConversionError

	c.Send(1)   // trigger ConversionError
	c.Send("1") // ProcessErrorLink will create a ProcessError
	c.Close()
	c.Wait()
	err := c.Error()

	assert.Error(t, err, "expected error from Strict chain")
}

func TestChain_NilLink(t *testing.T) {
	assert.PanicsWithValue(t, "link is <nil> in call to NewBase()", func() {
		chain.NewChain(
			basics.NewNilLink(),
		)
	})
}

func TestChain_ErrorConditions_Initialize(t *testing.T) {
	c := chain.NewChain(
		basics.NewErrorLink(cfg.WithArg("errorAt", "initialize")),
	)

	c.Send("123")
	c.Close()
	c.Wait()
	assert.ErrorContains(t, c.Error(), "initialize error")
}

func TestChain_ErrorConditions_Process(t *testing.T) {
	c := chain.NewChain(
		basics.NewErrorLink(cfg.WithArg("errorAt", "process")),
	)

	c.Send("123")
	c.Close()
	c.Wait()
	assert.ErrorContains(t, c.Error(), "process error: process error")
}

func TestChain_ErrorConditions_Complete(t *testing.T) {
	c := chain.NewChain(
		basics.NewErrorLink(cfg.WithArg("errorAt", "complete")),
	)

	c.Send("123")
	c.Close()
	c.Wait()
	assert.ErrorContains(t, c.Error(), "complete error")
}

func TestChain_ChainInsideChain(t *testing.T) {
	c := chain.NewChain(
		basics.NewChainInsideChain(),
		basics.NewStrLink(),
	).WithConfigs(
		cfg.WithArg("prefix", "test-prefix"),
	)

	c.Send("123")
	c.Close()

	received := []string{}
	for output, ok := chain.RecvAs[string](c); ok; output, ok = chain.RecvAs[string](c) {
		received = append(received, output)
	}

	require.NoError(t, c.Error())
	require.Len(t, received, 1)
	assert.Equal(t, "test-prefix123", received[0])
}

func TestChain_ChainInsideChainDirectArgs(t *testing.T) {
	c := chain.NewChain(
		basics.NewChainInsideChain(cfg.WithArg("prefix", "test-prefix")),
		basics.NewStrLink(),
	)

	c.Send("123")
	c.Close()

	received := []string{}
	for output, ok := chain.RecvAs[string](c); ok; output, ok = chain.RecvAs[string](c) {
		received = append(received, output)
	}

	require.NoError(t, c.Error())
	require.Len(t, received, 1)
	assert.Equal(t, "test-prefix123", received[0])
}

func TestChain_ChainDelay(t *testing.T) {
	w := &bytes.Buffer{}

	c := chain.NewChain(
		basics.NewDelayLink(),
		basics.NewStrLink(),
	).WithOutputters(
		output.NewWriterOutputter(),
	).WithConfigs(
		cfg.WithArg("writer", w),
		cfg.WithArg("delay", 1),
	)

	c.Send("123")
	c.Close()
	c.Wait()

	assert.Equal(t, "123\n", w.String())
	assert.NoError(t, c.Error())
}

func TestChain_WithAddedLinks(t *testing.T) {
	c := chain.NewChain(
		basics.NewStrLink(),
		basics.NewStrIntLink(),
	).WithAddedLinks(
		basics.NewIntLink(cfg.WithArg("intOp", func(i int) int { return i + 1 })),
	)

	c.Send("123")
	c.Close()

	received := []int{}
	for output, ok := chain.RecvAs[int](c); ok; output, ok = chain.RecvAs[int](c) {
		received = append(received, output)
	}

	assert.Equal(t, []int{124}, received)
}

func TestChain_ErrorOnLinkReuse(t *testing.T) {
	link := basics.NewStrLink()
	c1 := chain.NewChain(
		link,
	)

	c2 := chain.NewChain(
		link,
	)

	c1.Send("123")
	c1.Close()
	c1.Wait()

	c2.Send("123")
	c2.Close()
	c2.Wait()

	assert.NoError(t, c1.Error())
	assert.Error(t, c2.Error())
}

func TestChain_Logging(t *testing.T) {
	w := &bytes.Buffer{}
	c := chain.NewChain(
		basics.NewLoggingLink(),
	)
	c.WithName("test-chain")
	c.WithLogWriter(w)

	c.Send(basics.Msg{Level: slog.LevelInfo, Message: "test"})
	c.Close()
	c.Wait()

	assert.Contains(t, w.String(), "level=INFO link=test-chain/*basics.LoggingLink msg=test")
}

func TestChain_Logging_WithLogLevel(t *testing.T) {
	w := &bytes.Buffer{}
	c := chain.NewChain(
		basics.NewLoggingLink(),
	)
	c.WithLogWriter(w)
	c.WithName("test-chain")
	c.WithLogLevel(slog.LevelWarn)

	c.Send(basics.Msg{Level: slog.LevelDebug, Message: "Debug message"})
	c.Send(basics.Msg{Level: slog.LevelInfo, Message: "Info message"})
	c.Send(basics.Msg{Level: slog.LevelWarn, Message: "Warn message"})
	c.Send(basics.Msg{Level: slog.LevelError, Message: "Error message"})
	c.Close()
	c.Wait()

	formatted := strings.ReplaceAll(w.String(), "\\n", "\n")
	require.Contains(t, w.String(), "level=WARN link=test-chain/*basics.LoggingLink msg=\"Warn message\"", formatted)
	require.Contains(t, w.String(), "level=ERROR link=test-chain/*basics.LoggingLink msg=\"Error message\"", formatted)
	require.NotContains(t, w.String(), "level=DEBUG link=test-chain/*basics.LoggingLink msg=\"Debug message\"", formatted)
	require.NotContains(t, w.String(), "level=INFO link=test-chain/*basics.LoggingLink msg=\"Info message\"", formatted)
}

func TestChain_Permissions(t *testing.T) {
	c := chain.NewChain(
		basics.NewPermissionsLink().WithPermissions(cfg.NewPermission(cfg.AWS, "permission1")),
		basics.NewPermissionsLink().WithPermissions(cfg.NewPermission(cfg.GCP, "permission2")),
		basics.NewPermissionsLink().WithPermissions(cfg.NewPermission(cfg.Azure, "permission3")),
	)

	actual := []string{}
	for _, p := range c.Permissions() {
		actual = append(actual, p.String())
	}

	assert.Equal(t, []string{"AWS:permission1", "GCP:permission2", "Azure:permission3"}, actual)
}

func TestChain_PermissionsMap(t *testing.T) {
	c := chain.NewChain(
		basics.NewPermissionsLink().WithPermissions(
			cfg.NewPermission(cfg.AWS, "permissionA"),
			cfg.NewPermission(cfg.AWS, "permissionB"),
		),
		basics.NewPermissionsLink().WithPermissions(
			cfg.NewPermission(cfg.GCP, "permissionA"),
			cfg.NewPermission(cfg.GCP, "permissionB"),
		),
		basics.NewPermissionsLink().WithPermissions(
			cfg.NewPermission(cfg.Azure, "permissionA"),
			cfg.NewPermission(cfg.Azure, "permissionB"),
		),
	)

	assert.Len(t, c.PermissionsMap(), 3)

	for _, permissions := range c.PermissionsMap() {
		assert.Equal(t, permissions, []string{"permissionA", "permissionB"})
	}
}

func TestChain_Permissions_NoPermissions(t *testing.T) {
	c := chain.NewChain(
		basics.NewStrLink(),
		basics.NewStrLink(),
		basics.NewStrLink(),
	)

	perms := c.Permissions()
	assert.Empty(t, perms)
}
