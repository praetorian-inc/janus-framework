package cfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParam(t *testing.T) {
	param := NewParam[string]("test", "test param").WithDefault("default value")

	assert.Equal(t, "test", param.Name())
	assert.Equal(t, "test param", param.Description())
	assert.Equal(t, "default value", param.Value())
	assert.True(t, param.HasDefault())
	assert.False(t, param.Required())
	assert.Equal(t, "string", param.Type())

	param = param.AsRequired()
	assert.True(t, param.Required())
}

func TestParam_NoDefault(t *testing.T) {
	param := NewParam[string]("test", "test param")

	assert.Equal(t, "", param.Value(), "default method should return empty string")
	assert.False(t, param.HasDefault())
}

func TestParam_Flag(t *testing.T) {
	param := NewParam[string]("test", "test param").WithShortcode("t")
	noShortcode := NewParam[string]("test", "test param")

	assert.Equal(t, "test", param.Name())
	assert.Equal(t, "t", param.Shortcode())
	assert.Equal(t, "t", param.Flag())

	assert.Equal(t, "test", noShortcode.Name())
	assert.Equal(t, "", noShortcode.Shortcode())
	assert.Equal(t, "test", noShortcode.Flag())
}

func TestParam_ConvertFromCLIString(t *testing.T) {
	param := NewParam[[]int]("test", "test param")

	converted, err := param.convertFromCLIString("1,2,3")
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, converted)

	_, err = param.convertFromCLIString("invalid value")
	assert.EqualError(t, err, `error at index 0: strconv.Atoi: parsing "invalid value": invalid syntax`)
}
