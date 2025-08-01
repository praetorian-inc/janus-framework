package output_test

import (
	"bytes"
	"testing"

	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriterOutputter(t *testing.T) {
	writer := &bytes.Buffer{}
	outputter := output.NewWriterOutputter(cfg.WithArg("writer", writer))
	require.NotNil(t, outputter)

	err := outputter.Initialize()
	require.NoError(t, err)

	writerOutputter := outputter.(*output.WriterOutputter)
	err = writerOutputter.Output("test")
	assert.NoError(t, err)
	assert.Equal(t, "test\n", writer.String())
}
