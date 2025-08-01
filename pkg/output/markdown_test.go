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

type mockMarkdownable struct {
	column string
	row    int
	value  string
}

func (m *mockMarkdownable) Columns() []string {
	return []string{m.column}
}

func (m *mockMarkdownable) Rows() []int {
	return []int{m.row}
}

func (m *mockMarkdownable) Values() []any {
	return []any{m.value}
}

func TestMarkdownOutputter(t *testing.T) {
	outputter := output.NewMarkdownOutputter(cfg.WithArg("mdoutfile", "test.md"))
	require.NotNil(t, outputter, "received nil outputter")

	err := outputter.Initialize()
	require.NoError(t, err)

	markdownOutputter := outputter.(*output.MarkdownOutputter)

	i := 1
	for ; i < 5; i++ {
		dataA := &mockMarkdownable{column: "columnA", row: i, value: fmt.Sprintf("value%dA", i)}
		dataB := &mockMarkdownable{column: "columnB", row: i, value: fmt.Sprintf("value%dB", i)}

		require.NoError(t, markdownOutputter.Output(dataA))
		require.NoError(t, markdownOutputter.Output(dataB))
	}
	dataEnd := &mockMarkdownable{column: "columnA", row: i, value: "verylongvalue"}
	require.NoError(t, markdownOutputter.Output(dataEnd))

	outputter.Complete()

	content, err := os.ReadFile("test.md")
	require.NoError(t, err)

	expectedTable := `| columnA       | columnB |
| ------------- | ------- |
| value1A       | value1B |
| value2A       | value2B |
| value3A       | value3B |
| value4A       | value4B |
| verylongvalue |         |
`
	assert.Equal(t, expectedTable, string(content), fmt.Sprintf("expected:\n%s\nactual:\n%s", expectedTable, string(content)))

	os.Remove("test.md")
}

func TestMarkdownOutputter_NoRowsID(t *testing.T) {
	outputter := output.NewMarkdownOutputter(cfg.WithArg("mdoutfile", "test.md"))
	require.NotNil(t, outputter, "received nil outputter")

	err := outputter.Initialize()
	require.NoError(t, err)

	markdownOutputter := outputter.(*output.MarkdownOutputter)

	for i := 0; i < 4; i++ {
		dataA := &mockMarkdownable{column: "columnA", value: fmt.Sprintf("value%dA", i+1)}
		dataB := &mockMarkdownable{column: "columnB", value: fmt.Sprintf("value%dB", i+1)}

		require.NoError(t, markdownOutputter.Output(dataA))
		require.NoError(t, markdownOutputter.Output(dataB))
	}

	outputter.Complete()

	content, err := os.ReadFile("test.md")
	require.NoError(t, err)

	expectedTable := `| columnA | columnB |
| ------- | ------- |
| value1A | value1B |
| value2A | value2B |
| value3A | value3B |
| value4A | value4B |
`

	assert.Equal(t, expectedTable, string(content), fmt.Sprintf("expected:\n%s\nactual:\n%s", expectedTable, string(content)))

	os.Remove("test.md")
}

func TestMarkdownOutputter_NoColumns(t *testing.T) {
	outputter := output.NewMarkdownOutputter(cfg.WithArg("mdoutfile", "test.md"))
	require.NotNil(t, outputter, "received nil outputter")

	err := outputter.Initialize()
	require.NoError(t, err)

	markdownOutputter := outputter.(*output.MarkdownOutputter)

	for i := 0; i < 4; i++ {
		data := &mockMarkdownable{value: fmt.Sprintf("value%dA", i+1)}
		require.NoError(t, markdownOutputter.Output(data))
	}

	outputter.Complete()

	content, err := os.ReadFile("test.md")
	require.NoError(t, err)

	expectedTable := `| string  |
| ------- |
| value1A |
| value2A |
| value3A |
| value4A |
`

	assert.Equal(t, expectedTable, string(content), fmt.Sprintf("expected:\n%s\nactual:\n%s", expectedTable, string(content)))

	os.Remove("test.md")
}

type mockMultiMarkdownable struct {
	columns []string
	rows    []int
	values  []any
}

func (m *mockMultiMarkdownable) Columns() []string {
	return m.columns
}

func (m *mockMultiMarkdownable) Rows() []int {
	return m.rows
}

func (m *mockMultiMarkdownable) Values() []any {
	return m.values
}

func TestMarkdownOutputter_MultipleFields(t *testing.T) {
	outputter := output.NewMarkdownOutputter(cfg.WithArg("mdoutfile", "test.md"))
	require.NotNil(t, outputter, "received nil outputter")

	err := outputter.Initialize()
	require.NoError(t, err)

	markdownOutputter := outputter.(*output.MarkdownOutputter)

	i := 1
	for ; i < 5; i++ {
		dataA := &mockMultiMarkdownable{columns: []string{"columnA1", "columnA2"}, rows: []int{i, i}, values: []any{fmt.Sprintf("value%dA", i), fmt.Sprintf("value%dA2", i)}}
		dataB := &mockMultiMarkdownable{columns: []string{"columnB1", "columnB2"}, rows: []int{i, i}, values: []any{fmt.Sprintf("value%dB", i), fmt.Sprintf("value%dB2", i)}}

		require.NoError(t, markdownOutputter.Output(dataA))
		require.NoError(t, markdownOutputter.Output(dataB))
	}

	outputter.Complete()

	content, err := os.ReadFile("test.md")
	require.NoError(t, err)

	expectedTable := `| columnA1 | columnA2 | columnB1 | columnB2 |
| -------- | -------- | -------- | -------- |
| value1A  | value1A2 | value1B  | value1B2 |
| value2A  | value2A2 | value2B  | value2B2 |
| value3A  | value3A2 | value3B  | value3B2 |
| value4A  | value4A2 | value4B  | value4B2 |
`
	assert.Equal(t, expectedTable, string(content), fmt.Sprintf("expected:\n%s\nactual:\n%s", expectedTable, string(content)))

	os.Remove("test.md")
}

func TestMarkdownOutputter_MultipleFieldsWithMissings(t *testing.T) {
	outputter := output.NewMarkdownOutputter(cfg.WithArg("mdoutfile", "test.md"))
	require.NotNil(t, outputter, "received nil outputter")

	err := outputter.Initialize()
	require.NoError(t, err)

	markdownOutputter := outputter.(*output.MarkdownOutputter)

	i := 1
	for ; i < 5; i++ {
		dataA := &mockMultiMarkdownable{columns: []string{"columnA1", "columnA2"}, rows: []int{i, i}, values: []any{fmt.Sprintf("value%dA", i)}}
		dataB := &mockMultiMarkdownable{columns: []string{"columnB2"}, rows: []int{i, i}, values: []any{fmt.Sprintf("value%dB", i), fmt.Sprintf("value%dB2", i)}}

		require.NoError(t, markdownOutputter.Output(dataA))
		require.NoError(t, markdownOutputter.Output(dataB))
	}

	outputter.Complete()

	content, err := os.ReadFile("test.md")
	require.NoError(t, err)

	expectedTable := `| columnA1 | columnA2 | columnB2 | string   |
| -------- | -------- | -------- | -------- |
| value1A  |          | value1B  | value1B2 |
| value2A  |          | value2B  | value2B2 |
| value3A  |          | value3B  | value3B2 |
| value4A  |          | value4B  | value4B2 |
`
	assert.Equal(t, expectedTable, string(content), fmt.Sprintf("expected:\n%s\nactual:\n%s", expectedTable, string(content)))

	os.Remove("test.md")
}
