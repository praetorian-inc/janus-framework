package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckBinaryExists(t *testing.T) {
	assert.True(t, CheckBinaryExists("ls"))
	assert.False(t, CheckBinaryExists("nonexistent"))
}

func TestParsePrimative(t *testing.T) {
	stringString := "hello"
	stringResult, err := ParsePrimative[string](stringString)
	assert.NoError(t, err)
	assert.Equal(t, "hello", stringResult)

	boolString := "true"
	boolResult, err := ParsePrimative[bool](boolString)
	assert.NoError(t, err)
	assert.Equal(t, true, boolResult)

	intString := "123"
	intResult, err := ParsePrimative[int](intString)
	assert.NoError(t, err)
	assert.Equal(t, 123, intResult)

	stringSliceString := "hello,world"
	stringSliceResult, err := ParsePrimative[[]string](stringSliceString)
	assert.NoError(t, err)
	assert.Equal(t, []string{"hello", "world"}, stringSliceResult)

	intSliceString := "1,2,3"
	intSliceResult, err := ParsePrimative[[]int](intSliceString)
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, intSliceResult)
}

func TestIsConvertable(t *testing.T) {
	assert.True(t, IsConvertable[int]())
	assert.True(t, IsConvertable[[]int]())
	assert.False(t, IsConvertable[map[string]int]())
}
