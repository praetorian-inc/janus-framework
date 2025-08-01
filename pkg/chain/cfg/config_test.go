package cfg_test

import (
	"context"
	"os/exec"
	"testing"

	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

type mockParamable struct {
	*cfg.ContextHolder
	*cfg.ParamHolder
	*cfg.MethodsHolder
	*cfg.Logger
}

func (m *mockParamable) SetParams(params ...cfg.Param) error {
	return nil
}

func NewMockParamable() *mockParamable {
	return &mockParamable{
		ContextHolder: cfg.NewContextHolder(),
		ParamHolder:   cfg.NewParamHolder(),
		MethodsHolder: cfg.NewMethodsHolder(),
		Logger:        cfg.NewLogger(),
	}
}

func TestConfig_WithArg(t *testing.T) {
	paramable := NewMockParamable()
	paramable.ParamHolder = cfg.NewParamHolder()
	paramable.ParamHolder.SetParams(
		cfg.NewParam[string]("test", "test description"),
	)

	config := cfg.WithArg("test", "test value")
	err := config(paramable)

	assert.NoError(t, err)
	assert.Equal(t, "test value", paramable.Arg("test"))
}

func TestConfig_WithArgs(t *testing.T) {
	paramable := NewMockParamable()
	paramable.ParamHolder = cfg.NewParamHolder()
	paramable.ParamHolder.SetParams(
		cfg.NewParam[string]("test1", "test1 description"),
		cfg.NewParam[string]("test2", "test2 description"),
	)

	config := cfg.WithArgs(map[string]any{"test1": "test1 value", "test2": "test2 value"})
	err := config(paramable)

	assert.NoError(t, err)
	assert.Equal(t, "test1 value", paramable.Arg("test1"))
	assert.Equal(t, "test2 value", paramable.Arg("test2"))
}

func TestConfig_WithCLIArgs(t *testing.T) {
	paramable := NewMockParamable()
	paramable.ParamHolder = cfg.NewParamHolder()
	paramable.ParamHolder.SetParams(
		cfg.NewParam[string]("test", "test description"),
		cfg.NewParam[[]string]("test2", "test2 description"),
	)

	config := cfg.WithCLIArgs([]string{"-test", "test value", "-test2", "valueA,valueB,valueC"})
	config(paramable)

	assert.Equal(t, "test value", paramable.Arg("test"))
	assert.Equal(t, []string{"valueA", "valueB", "valueC"}, paramable.Arg("test2"))
}

func TestConfig_WithMethods(t *testing.T) {
	paramable := NewMockParamable()

	didExecute := false
	methods := &testutils.MockMethods{ExecuteFn: func(cmd *exec.Cmd, lineparser func(string)) error {
		didExecute = true
		return nil
	}}

	config := cfg.WithMethods(methods)
	config(paramable)
	paramable.MethodsHolder.ExecuteCmd(exec.Command("echo", "test"), func(_ string) {})

	assert.True(t, didExecute)
}

func TestConfig_WithContext(t *testing.T) {
	paramable := NewMockParamable()

	ctx := context.WithValue(context.Background(), "test", "test value")
	config := cfg.WithContext(ctx)
	config(paramable)

	assert.Equal(t, "test value", paramable.Context().Value("test"))
}
