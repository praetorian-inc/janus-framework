package output_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONOutputter(t *testing.T) {
	outputter := output.NewJSONOutputter(cfg.WithArg("jsonoutfile", "test.json"))
	require.NotNil(t, outputter, "received nil outputter")

	err := outputter.Initialize()
	require.NoError(t, err)

	o := outputter.(*output.JSONOutputter)
	require.NoError(t, o.Output(map[string]string{"test": "foobar"}))
	require.NoError(t, o.Complete())

	content, err := os.ReadFile("test.json")
	assert.NoError(t, err)
	expected := "[{\"test\":\"foobar\"}]\n"
	assert.Equal(t, expected, string(content), fmt.Sprintf("expected %s, got %s", expected, string(content)))

	os.Remove("test.json")
}

func TestJSONOutputter_List(t *testing.T) {
	outputter := output.NewJSONOutputter(cfg.WithArg("jsonoutfile", "test.json"))
	require.NotNil(t, outputter, "received nil outputter")

	err := outputter.Initialize()
	require.NoError(t, err)

	o := outputter.(*output.JSONOutputter)
	require.NoError(t, o.Output(map[string]string{"test": "foobar"}))
	require.NoError(t, o.Output(map[string]string{"test": "bazbat"}))
	require.NoError(t, o.Complete())

	content, err := os.ReadFile("test.json")
	assert.NoError(t, err)
	assert.Equal(t, "[{\"test\":\"foobar\"},{\"test\":\"bazbat\"}]\n", string(content))

	os.Remove("test.json")
}

type mockMarshaler struct {
	Value string
}

func (m *mockMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("{\"testABC\": \"%s\"}", m.Value)), nil
}

func TestJSONOutputter_Marshaler(t *testing.T) {
	outputter := output.NewJSONOutputter(cfg.WithArg("jsonoutfile", "test.json"))
	require.NotNil(t, outputter, "received nil outputter")

	err := outputter.Initialize()
	require.NoError(t, err)

	jsonOutputter := outputter.(*output.JSONOutputter)
	require.NoError(t, jsonOutputter.Output(&mockMarshaler{Value: "foobar"}))     // should print prefix
	require.NoError(t, jsonOutputter.Output(map[string]string{"test": "foobar"})) // should not print prefix
	require.NoError(t, jsonOutputter.Complete())
	content, err := os.ReadFile("test.json")
	assert.NoError(t, err)
	assert.Equal(t, "[{\"testABC\":\"foobar\"},{\"test\":\"foobar\"}]\n", string(content))

	os.Remove("test.json")
}
