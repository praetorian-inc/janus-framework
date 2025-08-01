package chain_test

import (
	"reflect"
	"testing"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cherrors"
	"github.com/praetorian-inc/janus-framework/pkg/testutils/mocks/basics"
	"github.com/praetorian-inc/janus-framework/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlue_Process(t *testing.T) {
	link := basics.NewStrLink()
	go chain.Process(link, "123")

	received, ok := chain.RecvAs[string](link)
	assert.True(t, ok)
	assert.Equal(t, "123", received)
}

func TestGlue_CopyFields(t *testing.T) {
	source := struct {
		Str        string
		Additional string
	}{Str: "123"}

	destination := struct {
		Str string
	}{}

	chain.CopyFields(reflect.ValueOf(source), reflect.ValueOf(&destination))

	assert.Equal(t, "123", destination.Str)
}

func TestGlue_CopyFields_Unexported(t *testing.T) {
	source := basics.NewEvenMoreThanStrStruct("123", "456", "789") // contains unexported field "hidden"
	destination := basics.EvenMoreThanStrStruct{}

	require.NotPanics(t, func() {
		chain.CopyFields(reflect.ValueOf(source), reflect.ValueOf(&destination))
	})

	assert.Equal(t, "123", destination.Str)
	assert.Equal(t, "456", destination.Additional)
}

func TestGlue_ConvertForLink(t *testing.T) {
	tests := []struct {
		explanation string
		link        chain.Link
		input       any
		expectError bool
	}{
		{
			explanation: "structs are not compatible",
			link:        basics.NewStrStructLink(),
			input:       basics.Mockable{MockMsg: "mocking"},
			expectError: true,
		},
		{
			explanation: "structs are compatible",
			link:        basics.NewStrStructLink(),
			input:       basics.MoreThanStrStruct{Str: "123", Additional: "456"},
			expectError: false,
		},
		{
			explanation: "string is not a struct",
			link:        basics.NewStrStructLink(),
			input:       "123",
			expectError: true,
		},
		{
			explanation: "glue code should handle pointers",
			link:        basics.NewStrStructLink(),
			input:       &basics.MoreThanStrStruct{Str: "123", Additional: "456"},
			expectError: false,
		},
		{
			explanation: "I/O types match",
			link:        basics.NewStrLink(),
			input:       "string",
			expectError: false,
		},
		{
			explanation: "interface is compatible",
			link:        basics.NewInterfaceLink(),
			input:       &basics.Mockable{MockMsg: "mocking"},
			expectError: false,
		},
		{
			explanation: "interface is compatible with reference to struct",
			link:        basics.NewInterfaceLink(),
			input:       basics.Mockable{MockMsg: "mocking"},
			expectError: false,
		},
		{
			explanation: "interface is compatible with struct",
			link:        basics.NewInterfaceLink(),
			input:       basics.StructMockable{Msg: "mocking"},
			expectError: false,
		},
		{
			explanation: "interface is compatible with struct, after dereferencing",
			link:        basics.NewInterfaceLink(),
			input:       &basics.StructMockable{Msg: "mocking"},
			expectError: false,
		},
		{
			explanation: "interface{} is allowed",
			link:        basics.NewEchoLink(),
			input:       "hello there",
			expectError: false,
		},
		{
			explanation: "nil causes error instead of panic",
			link:        basics.NewEchoLink(),
			input:       nil,
			expectError: true,
		},
		{
			explanation: "nil causes error instead of panic (mismatched types)",
			link:        basics.NewStrLink(),
			input:       nil,
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.explanation, func(t *testing.T) {
			_, err := chain.ConvertForLink(test.input, test.link)
			if test.expectError {
				assert.Error(t, err, test.explanation+" (should fail)")
			} else {
				assert.NoError(t, err, test.explanation+" (should succeed)")
			}
		})
	}
}

func TestGlue_ProcessWithConversion(t *testing.T) {
	link := basics.NewStrStructLink()
	go chain.Process(link, basics.MoreThanStrStruct{Str: "123", Additional: "456"})

	received, ok := chain.RecvAs[string](link)
	assert.True(t, ok)
	assert.Equal(t, "123", received)
}

func TestGlue_Convert(t *testing.T) {
	tests := []struct {
		explanation string
		input       any
		outputType  reflect.Type
		expectError bool
	}{
		{
			explanation: "string is compatible with string",
			input:       "hello",
			outputType:  reflect.TypeOf("string"),
			expectError: false,
		},
		{
			explanation: "int is compatible with int",
			input:       123,
			outputType:  reflect.TypeOf(1),
			expectError: false,
		},
		{
			explanation: "input satisfies output interface",
			input:       &basics.Mockable{MockMsg: "mocking"},
			outputType:  reflect.TypeOf((*basics.Mockable)(nil)),
			expectError: false,
		},
		{
			explanation: "input has every field needed for output",
			input:       basics.MoreThanStrStruct{Str: "123", Additional: "456"},
			outputType:  reflect.TypeOf(struct{ Str string }{}),
			expectError: false,
		},
		{
			explanation: "input does not have every field needed for output",
			input:       struct{ Str string }{Str: "123"},
			outputType:  reflect.TypeOf(basics.MoreThanStrStruct{}),
			expectError: true,
		},
		{
			explanation: "glue code should handle pointers",
			input:       &types.ScannableAsset{Target: "target"},
			outputType:  reflect.TypeOf(&types.ScannableAsset{}),
			expectError: false,
		},
		{
			explanation: "glue code should handle pointer differences (struct input, ptr output)",
			input:       types.ScannableAsset{Target: "target"},
			outputType:  reflect.TypeOf(&types.ScannableAsset{}),
			expectError: false,
		},
		{
			explanation: "glue code should handle pointer differences (ptr input, struct output)",
			input:       &types.ScannableAsset{Target: "target"},
			outputType:  reflect.TypeOf(types.ScannableAsset{}),
			expectError: false,
		},
		{
			explanation: "glue code should handle pointer differences for compatible structs (ptr input, struct output)",
			input:       &basics.MoreThanStrStruct{Str: "123", Additional: "456"},
			outputType:  reflect.TypeOf(struct{ Str string }{}),
			expectError: false,
		},
		{
			explanation: "glue code should handle pointer differences for compatible structs (struct input, ptr output)",
			input:       basics.MoreThanStrStruct{Str: "123", Additional: "456"},
			outputType:  reflect.TypeOf(&struct{ Str string }{}),
			expectError: false,
		},
		{
			explanation: "glue code should handle pointers for compatible structs",
			input:       &basics.MoreThanStrStruct{Str: "123", Additional: "456"},
			outputType:  reflect.TypeOf(&struct{ Str string }{}),
			expectError: false,
		},
		{
			explanation: "glue code should not handle pointers for incompatible structs (ptr to ptr)",
			input:       &struct{ Str string }{Str: "123"},
			outputType:  reflect.TypeOf(&types.ScannableAsset{}),
			expectError: true,
		},
		{
			explanation: "glue code should not handle pointers for incompatible structs (ptr to struct)",
			input:       &struct{ Str string }{Str: "123"},
			outputType:  reflect.TypeOf(types.ScannableAsset{}),
			expectError: true,
		},
		{
			explanation: "glue code should not handle pointers for incompatible structs (struct to ptr)",
			input:       struct{ Str string }{Str: "123"},
			outputType:  reflect.TypeOf(&types.ScannableAsset{}),
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.explanation, func(t *testing.T) {
			_, err := chain.Convert(test.input, test.outputType)
			if test.expectError {
				assert.Error(t, err, test.explanation+" (should fail)")
				_, isConversionError := err.(*cherrors.ConversionError)
				assert.True(t, isConversionError, "expected conversion error: %T (%v)", err, err)
			} else {
				assert.NoError(t, err, test.explanation+" (should succeed)")
			}
		})
	}
}

func TestGlue_Errors(t *testing.T) {
	_, conversionError := chain.Convert("not an integer", reflect.TypeOf(1))
	_, isConversionError := conversionError.(*cherrors.ConversionError)
	assert.Error(t, conversionError)
	assert.True(t, isConversionError, "expected conversion error")

	processError := chain.Process(basics.NewProcessErrorLink(), "123")
	_, isProcessError := processError.(*cherrors.ProcessError)
	assert.Error(t, processError)
	assert.True(t, isProcessError, "expected process error: %T (%v)", processError, processError)
}

func TestGlue_ConvertForJSON(t *testing.T) {
	jsonData := `{"str": "123", "additional": "456"}`

	converted, err := chain.ConvertForJSON(jsonData, basics.NewMoreThanStrStructLink())

	assert.NoError(t, err)
	assert.Equal(t, &basics.MoreThanStrStruct{Str: "123", Additional: "456"}, converted)
}
