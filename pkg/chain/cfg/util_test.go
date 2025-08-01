package cfg

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArg_As(t *testing.T) {
	tests := []struct {
		input       any
		expectError bool
		expected    any
	}{
		{input: "hello", expectError: false, expected: "hello"},
		{input: 123, expectError: false, expected: 123},
		{input: true, expectError: false, expected: true},
		{input: []string{"hello", "world"}, expectError: false, expected: []string{"hello", "world"}},
		{input: []int{1, 2}, expectError: false, expected: []int{1, 2}},
		{input: []bool{true, false}, expectError: false, expected: []bool{true, false}},
		{input: map[string]any{"hello": "world"}, expectError: false, expected: map[string]any{"hello": "world"}},
		{input: chan any(nil), expectError: false, expected: chan any(nil)},
	}

	for _, test := range tests {
		actual, err := As[any](test.input)
		if test.expectError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, test.expected, actual)
		}
	}
}

func TestArg_CanBeNil(t *testing.T) {
	assert.True(t, canBeNil[[]string]())
	assert.True(t, canBeNil[[]int]())
	assert.True(t, canBeNil[chan any]())
	assert.True(t, canBeNil[map[string]any]())
	assert.False(t, canBeNil[string]())
	assert.False(t, canBeNil[int]())
	assert.False(t, canBeNil[bool]())
	assert.False(t, canBeNil[struct{ data string }]())
}

func TestArg_IsInterface(t *testing.T) {
	assert.True(t, isInterface[io.Writer]())
	assert.False(t, isInterface[string]())
}

func TestArg_IsSettable(t *testing.T) {
	assert.True(t, isSettable[string]("string value"))
	assert.True(t, isSettable[int](1))
	assert.True(t, isSettable[bool](true))
	assert.True(t, isSettable[[]string]([]string{"hello", "world"}))
	assert.True(t, isSettable[[]int]([]int{1, 2}))
	assert.True(t, isSettable[[]bool]([]bool{true, false}))
	assert.True(t, isSettable[map[string]any](map[string]any{"hello": "world"}))
	assert.True(t, isSettable[chan any](make(chan any)))
	assert.True(t, isSettable[struct{ data string }](struct{ data string }{"hello"}))
	assert.True(t, isSettable[io.Writer](os.Stdout))

	assert.False(t, isSettable[string](1))
	assert.False(t, isSettable[string](nil))
	assert.False(t, isSettable[string]([]string{"hello", "world"}))
	assert.False(t, isSettable[int]([]int{1, 2}))
	assert.False(t, isSettable[int]("hello"))
	assert.False(t, isSettable[io.Writer](nil))
	assert.False(t, isSettable[io.Writer]([]string{"hello", "world"}))
	assert.False(t, isSettable[io.Writer](map[string]any{"hello": "world"}))
	assert.False(t, isSettable[io.Writer](struct{ data string }{"hello"}))
}

func TestArg_GetInterfaceType(t *testing.T) {
	assert.Equal(t, "io.Writer", getInterfaceType[io.Writer]().String())
	assert.Equal(t, "io.Reader", getInterfaceType[io.Reader]().String())
}
