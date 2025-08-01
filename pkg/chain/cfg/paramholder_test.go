package cfg_test

import (
	"io"
	"testing"

	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParamHolder(t *testing.T) {
	params := []cfg.Param{
		cfg.NewParam[string]("test", "test").WithDefault("default value"),
	}

	holder := cfg.NewParamHolder()
	holder.SetParams(params...)

	assert.True(t, holder.HasParam("test"))
	assert.False(t, holder.HasParam("nottest"))
}

func TestParamHolder_SetParam(t *testing.T) {
	holder := cfg.NewParamHolder()

	err := holder.SetParam(cfg.NewParam[string]("test", "test"))
	assert.NoError(t, err)

	assert.True(t, holder.HasParam("test"))
}

func TestParamHolder_Arg(t *testing.T) {
	holder := cfg.NewParamHolder()
	holder.SetParams(
		cfg.NewParam[string]("string value", "test"),
		cfg.NewParam[string]("string value with default", "test2").WithDefault("default value"),
		cfg.NewParam[int]("int value", "test3"),
	)

	err1 := holder.SetArg("string value", "hello")
	err2 := holder.SetArg("int value", 123)

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	assert.Equal(t, "hello", holder.Arg("string value"))
	assert.Equal(t, "default value", holder.Arg("string value with default"))
	assert.Equal(t, 123, holder.Arg("int value"))
}

func TestParamHolder_SetArgsFromList(t *testing.T) {
	holder := cfg.NewParamHolder()

	err := holder.SetParams(
		cfg.NewParam[string]("string value", "test").WithShortcode("s"),
		cfg.NewParam[string]("string value with default", "test2").WithDefault("default value").WithShortcode("s2"),
		cfg.NewParam[int]("int value", "test3").WithShortcode("i"),
		cfg.NewParam[[]string]("string slice value", "test4").WithShortcode("ss"),
		cfg.NewParam[bool]("noshortcode", "test5"),
	)
	require.NoError(t, err)

	err = holder.SetArgsFromList([]string{
		"-s", "hello",
		"-i", "123",
		"-ss", "hello,world",
		"-noshortcode", "true",
	})
	require.NoError(t, err)

	assert.Equal(t, "hello", holder.Arg("string value"))
	assert.Equal(t, "default value", holder.Arg("string value with default"))
	assert.Equal(t, 123, holder.Arg("int value"))
	assert.Equal(t, []string{"hello", "world"}, holder.Arg("string slice value"))
	assert.Equal(t, true, holder.Arg("noshortcode"))
}

func TestParamHolder_SetArgsFromList_Reverse(t *testing.T) {
	holder := cfg.NewParamHolder()

	err := holder.SetArgsFromList([]string{
		"-s", "hello",
		"-i", "123",
		"-ss", "hello,world",
		"-noshortcode", "true",
	})
	require.NoError(t, err)

	err = holder.SetParams(
		cfg.NewParam[string]("string value", "test").WithShortcode("s"),
		cfg.NewParam[string]("string value with default", "test2").WithDefault("default value").WithShortcode("s2"),
		cfg.NewParam[int]("int value", "test3").WithShortcode("i"),
		cfg.NewParam[[]string]("string slice value", "test4").WithShortcode("ss"),
		cfg.NewParam[bool]("noshortcode", "test5"),
	)
	require.NoError(t, err)

	assert.Equal(t, "hello", holder.Arg("string value"))
	assert.Equal(t, "default value", holder.Arg("string value with default"))
	assert.Equal(t, 123, holder.Arg("int value"))
	assert.Equal(t, []string{"hello", "world"}, holder.Arg("string slice value"))
	assert.Equal(t, true, holder.Arg("noshortcode"))

}

func TestParamHolder_Args(t *testing.T) {
	holder := cfg.NewParamHolder()
	holder.SetParams(
		cfg.NewParam[string]("string value", ""),
		cfg.NewParam[string]("string value with default", "").WithDefault("default value"),
		cfg.NewParam[int]("int value", ""),
		cfg.NewParam[[]string]("string slice value", ""),
		cfg.NewParam[bool]("bool value", ""),
	)

	holder.SetArg("string value", "hello")
	holder.SetArg("int value", 123)
	holder.SetArg("bool value", true)

	args := holder.Args()

	require.Len(t, args, 4)

	assert.Equal(t, "hello", args["string value"])
	assert.Equal(t, "default value", args["string value with default"])
	assert.Equal(t, 123, args["int value"])
	assert.Nil(t, args["string slice value"])
	assert.Equal(t, true, args["bool value"])
}

func TestParamHolder_InvalidParams(t *testing.T) {
	holder := cfg.NewParamHolder()

	err := holder.SetParam(cfg.NewParam[io.Writer]("missing converter", "").WithShortcode("w"))
	assert.EqualError(t, err, `invalid shortcode for param "missing converter": converter required to use shortcode for type "io.Writer"`)

	err = holder.SetParam(cfg.NewParam[string]("duplicate parameter name", "description 1").WithShortcode("t"))
	assert.NoError(t, err)
	err = holder.SetParam(cfg.NewParam[string]("duplicate parameter name", "description 2").WithShortcode("t"))
	assert.EqualError(t, err, "param already exists with name \"duplicate parameter name\"")

}
