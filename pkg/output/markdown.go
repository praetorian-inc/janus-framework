package output

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type Markdownable interface {
	Columns() []string
	Rows() []int
	Values() []any
}

type Sorter func(a, b string) bool

type mdColumn struct {
	name   string
	values []string
	width  int
}

type mdCell struct {
	column string
	row    int
	value  any
}

type MarkdownOutputter struct {
	*chain.BaseOutputter
	table               map[string]mdColumn
	outfile             string
	sorter              Sorter
	longestColumnLength int
}

func NewMarkdownOutputter(configs ...cfg.Config) chain.Outputter {
	m := &MarkdownOutputter{table: make(map[string]mdColumn)}
	m.BaseOutputter = chain.NewBaseOutputter(m, configs...)
	return m
}

func (m *MarkdownOutputter) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("mdoutfile", "the file to write the markdown to").WithDefault("out.md"),
		cfg.NewParam[[]string]("columns", "the columns to write to the markdown"),
		cfg.NewParam[Sorter]("sorter", "sorter function to sort the columns (defaults to alphabetical)").WithDefault(func(a, b string) bool { return a < b }),
	}
}

func (m *MarkdownOutputter) Initialize() error {
	columns, err := cfg.As[[]string](m.Arg("columns"))
	if err == nil {
		for _, column := range columns {
			m.table[column] = mdColumn{name: column, values: []string{}}
		}
	}

	outfile, err := cfg.As[string](m.Arg("mdoutfile"))
	if err != nil {
		return fmt.Errorf("error getting mdoutfile: %w", err)
	}
	m.outfile = outfile

	sorter, err := cfg.As[Sorter](m.Arg("sorter"))
	if err != nil {
		sorter = func(a, b string) bool {
			return a < b
		}
	}
	m.sorter = sorter

	return nil
}

func (m *MarkdownOutputter) Output(mdData Markdownable) error {
	columns := mdData.Columns()
	rows := mdData.Rows()
	values := mdData.Values()

	cells := []mdCell{}
	for i := 0; i < max(len(columns), len(rows), len(values)); i++ {
		cell := mdCell{}
		if i < len(columns) {
			cell.column = columns[i]
		}
		if i < len(rows) {
			cell.row = rows[i]
		}
		if i < len(values) {
			cell.value = values[i]
		}
		cells = append(cells, cell)
	}

	for _, cell := range cells {
		err := m.processColumn(cell.column, cell.row, cell.value)
		if err != nil {
			return fmt.Errorf("error processing column: %w", err)
		}
	}

	return nil
}

func (m *MarkdownOutputter) processColumn(columnName string, rowID int, cellItem any) error {
	if columnName == "" && cellItem != nil {
		columnName = fmt.Sprintf("%T", cellItem)
	} else if columnName == "" && cellItem == nil {
		return fmt.Errorf("column name is empty and cell item is nil")
	}

	cellData := ""
	stringer, ok := cellItem.(fmt.Stringer)
	if ok {
		cellData = stringer.String()
	} else if cellItem == nil {
		cellData = ""
	} else {
		cellData = fmt.Sprintf("%v", cellItem)
	}

	column := m.table[columnName]
	column.name = columnName
	column.width = max(column.width, len(cellData), len(columnName))

	for len(column.values) < rowID {
		column.values = append(column.values, "")
	}
	if rowID == 0 {
		column.values = append(column.values, cellData)
	} else {
		column.values[rowID-1] = cellData
	}
	m.longestColumnLength = max(m.longestColumnLength, len(column.values))

	m.table[columnName] = column

	return nil
}

func (m *MarkdownOutputter) Complete() error {
	columns := []mdColumn{}
	for _, column := range m.table {
		columns = append(columns, column)
	}

	sort.Slice(columns, func(i, j int) bool {
		return m.sorter(columns[i].name, columns[j].name)
	})

	rows := make([]string, m.longestColumnLength+2) // +1 for header, +1 for separator
	for _, column := range columns {
		rows[0] += fmt.Sprintf("| %-*s ", column.width, column.name)
		rows[1] += fmt.Sprintf("| %-s ", strings.Repeat("-", column.width))
		for len(column.values) < m.longestColumnLength {
			column.values = append(column.values, "")
		}
		for i, value := range column.values {
			rows[i+2] += fmt.Sprintf("| %-*s ", column.width, value)
		}
	}

	table := strings.Join(rows, "|\n") + "|\n"

	writer, err := os.Create(m.outfile)
	if err != nil {
		return fmt.Errorf("error creating markdown file: %w", err)
	}
	defer writer.Close()

	writer.WriteString(table)

	return nil
}
