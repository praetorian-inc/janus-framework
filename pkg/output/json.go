package output

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type JSONOutputter struct {
	*chain.BaseOutputter
	encoder *json.Encoder
	indent  int
	output  []any
}

func NewJSONOutputter(configs ...cfg.Config) chain.Outputter {
	j := &JSONOutputter{}
	j.BaseOutputter = chain.NewBaseOutputter(j, configs...)
	return j
}

func (j *JSONOutputter) Initialize() error {
	filename, err := cfg.As[string](j.Arg("jsonoutfile"))
	if err != nil {
		return fmt.Errorf("error getting jsonoutfile: %w", err)
	}

	indent, err := cfg.As[int](j.Arg("indent"))
	if err != nil {
		indent = 0
	}
	j.indent = indent

	slog.Debug("creating JSON output file", "filename", filename)
	writer, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating JSON: %w", err)
	}

	j.encoder = json.NewEncoder(writer)
	j.encoder.SetIndent("", strings.Repeat(" ", j.indent))

	return nil
}

func (j *JSONOutputter) Output(val any) error {
	j.output = append(j.output, val)
	return nil
}

func (j *JSONOutputter) Complete() error {
	return j.encoder.Encode(j.output)
}

func (j *JSONOutputter) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("jsonoutfile", "the file to write the JSON to").WithDefault("out.json"),
		cfg.NewParam[int]("indent", "the number of spaces to use for the JSON indentation").WithDefault(0),
	}
}
