package chain_test

import (
	"bytes"
	"testing"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/output"
	"github.com/praetorian-inc/janus-framework/pkg/testutils/mocks/basics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModule(t *testing.T) {
	w := &bytes.Buffer{}

	module := chain.NewModule(
		cfg.NewMetadata(
			"test",
			"test",
		).WithChainInputParam("strings"),
	).WithLinks(
		basics.NewStrLink,
		basics.NewStrIntLink,
		basics.NewIntLink,
	).WithConfigs(
		cfg.WithArg("writer", w),
	).WithInputParam(
		cfg.NewParam[[]string]("strings", "strings to process"),
	).WithOutputters(
		output.NewWriterOutputter,
	)

	params := module.Params()
	paramsMap := make(map[string]cfg.Param)
	for _, param := range params {
		paramsMap[param.Name()] = param
	}

	require.Equal(t, 4, len(params))
	assert.NotNil(t, paramsMap["strings"])
	assert.NotNil(t, paramsMap["writer"])
	assert.NotNil(t, paramsMap["strOp"])
	assert.NotNil(t, paramsMap["intOp"])

	err := module.Run(cfg.WithCLIArgs([]string{"-strings", "1,2,3,4,5"}))
	assert.NoError(t, err)
	assert.NoError(t, module.Error())

	assert.Equal(t, "1\n2\n3\n4\n5\n", w.String())
}

func TestModule_Error(t *testing.T) {
	w := &bytes.Buffer{}

	errOnRunModule := chain.NewModule(
		cfg.NewMetadata(
			"test",
			"test",
		).WithChainInputParam("strings"),
	).WithLinks(
		basics.NewStrLink,
		basics.NewStrIntLink,
		basics.NewIntLink,
	).WithConfigs(
		cfg.WithArg("writer", w),
	).WithInputParam(
		cfg.NewParam[string]("strings", "strings to process"),
	).WithOutputters(
		output.NewWriterOutputter,
	)

	errDuringProcessModule := chain.NewModule(
		cfg.NewMetadata(
			"test",
			"test",
		).WithChainInputParam("strings"),
	).WithLinks(
		chain.ConstructLinkWithConfigs(basics.NewErrorLink, cfg.WithArg("errorAt", "process")),
	).WithConfigs(
		cfg.WithArg("writer", w),
	).WithInputParam(
		cfg.NewParam[[]string]("strings", "strings to process"),
	).WithOutputters(
		output.NewWriterOutputter,
	)

	errOnRunModule.Run(cfg.WithCLIArgs([]string{"-strings", "1,2,3,4,5"}))
	assert.ErrorContains(t, errOnRunModule.Error(), `module input parameter "strings"`)

	errDuringProcessModule.Run(cfg.WithCLIArgs([]string{"-strings", "1,2,3,4,5"}))
	assert.ErrorContains(t, errDuringProcessModule.Error(), "process error")
}

func TestModule_WithAddedLinks(t *testing.T) {
	w := &bytes.Buffer{}

	module := chain.NewModule(
		cfg.NewMetadata(
			"test",
			"test",
		).WithChainInputParam("strings"),
	).WithLinks(
		basics.NewStrLink,
		basics.NewStrIntLink,
	).WithConfigs(
		cfg.WithArg("writer", w),
	).WithInputParam(
		cfg.NewParam[[]string]("strings", "strings to process"),
	).WithOutputters(
		output.NewWriterOutputter,
	)

	module.WithAddedLinks(
		chain.ConstructLinkWithConfigs(basics.NewIntLink, cfg.WithArg("intOp", func(i int) int { return i * 2 })),
	)

	module.Run(cfg.WithCLIArgs([]string{"-strings", "1,2,3,4,5"}))

	assert.NoError(t, module.Error())
	assert.Equal(t, "2\n4\n6\n8\n10\n", w.String())
}

func TestModule_DefaultParamOverride(t *testing.T) {
	t.Skip()

	module := chain.NewModule(
		cfg.NewMetadata(
			"test",
			"test",
		),
	).WithLinks(
		basics.NewParamsLink,
		basics.NewStrLink,
	).WithInputParam(
		cfg.NewParam[int]("default", "default param").WithDefault(10),
	)

	c := module.New()

	assert.Equal(t, 10, c.Arg("default"), "modules may override default values")
	assert.NoError(t, module.Error(), "modules may override default values")
}

func TestModule_RequiredParamOverride(t *testing.T) {
	t.Skip()

	module := chain.NewModule(
		cfg.NewMetadata(
			"test",
			"test",
		),
	).WithLinks(
		basics.NewParamsLink,
		basics.NewStrLink,
	).WithInputParam(
		cfg.NewParam[string]("required", "required param"), // missing .AsRequired()
	)

	c := module.New()

	assert.Error(t, c.Error(), "modules may not override required values")
}

func TestModule_Reuse(t *testing.T) {
	module := chain.NewModule(
		cfg.NewMetadata(
			"test",
			"test",
		),
	).WithLinks(
		basics.NewStrIntLink,
	).WithOutputters(
		output.NewWriterOutputter,
	)

	c1 := module.New()
	c2 := module.New()

	w1 := &bytes.Buffer{}
	w2 := &bytes.Buffer{}

	c1.WithConfigs(cfg.WithArg("writer", w1))
	c2.WithConfigs(cfg.WithArg("writer", w2))

	for i := 0; i < 5; i++ {
		c1.Send("1")
		c2.Send("2")
	}
	c1.Close()
	c2.Close()
	c1.Wait()
	c2.Wait()

	assert.NoError(t, c1.Error())
	assert.NoError(t, c2.Error())

	assert.Equal(t, "1\n1\n1\n1\n1\n", w1.String())
	assert.Equal(t, "2\n2\n2\n2\n2\n", w2.String())
}

func TestModule_AutoRun(t *testing.T) {
	w := &bytes.Buffer{}

	link := basics.NewStrLink(cfg.WithArg("strOp", func(s string) string { return s + "!" }))

	module := chain.NewModule(
		cfg.NewMetadata(
			"AutoRun",
			"AutoRun test",
		),
	).WithLinks(
		func(...cfg.Config) chain.Link { return link },
	).WithConfigs(
		cfg.WithArg("writer", w),
	).WithOutputters(
		output.NewWriterOutputter,
	).WithAutoRun()

	module.Run()

	assert.NoError(t, module.Error())
	assert.Equal(t, "autorun!\n", w.String())
}

func TestModule_WithStrictness_Lax(t *testing.T) {
	w := &bytes.Buffer{}

	laxModule := chain.NewModule(
		cfg.NewMetadata(
			"strictness-lax-test",
			"Test module with Lax strictness",
		).WithChainInputParam("strings"),
	).WithStrictness(chain.Lax).
		WithLinks(
			basics.NewStrLink,
			basics.NewProcessErrorLink,
		).WithConfigs(
		cfg.WithArg("writer", w),
	).WithOutputters(
		output.NewWriterOutputter,
	).WithInputParam(
		cfg.NewParam[[]string]("strings", "strings to process"),
	)

	err := laxModule.Run(cfg.WithCLIArgs([]string{"-strings", "test"}))
	assert.NoError(t, err, "Lax strictness should not error on ProcessError")
	assert.NoError(t, laxModule.Error())
}

func TestModule_WithStrictness_Moderate(t *testing.T) {
	w := &bytes.Buffer{}

	moderateModule := chain.NewModule(
		cfg.NewMetadata(
			"strictness-moderate-test",
			"Test module with Moderate strictness",
		).WithChainInputParam("strings"),
	).WithLinks(
		basics.NewStrLink,
		basics.NewProcessErrorLink,
	).WithConfigs(
		cfg.WithArg("writer", w),
	).WithOutputters(
		output.NewWriterOutputter,
	).WithInputParam(
		cfg.NewParam[[]string]("strings", "strings to process"),
	)

	err := moderateModule.Run(cfg.WithCLIArgs([]string{"-strings", "test"}))
	assert.Error(t, err, "Moderate strictness should error on ProcessError")
	assert.Error(t, moderateModule.Error())
}
