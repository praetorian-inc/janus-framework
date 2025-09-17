package noseyparker

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/types"
)

type noseyParker struct {
	executeFn func(cmd *exec.Cmd, parser func(line string)) error
	datastore string
	done      chan bool
}

func newNoseyParker(done chan bool, executeFn func(cmd *exec.Cmd, parser func(line string)) error) *noseyParker {
	np := &noseyParker{done: done, executeFn: executeFn}
	return np
}

func (n *noseyParker) startNpScanner() *io.PipeWriter {
	pipeReader, pipeWriter := io.Pipe()

	go func() {
		defer func() {
			n.done <- true
		}()

		scanCmd := exec.Command(
			"noseyparker",
			"--verbose",
			"scan",
			"--datastore", n.datastore,
			"--progress", "never",
			"--enumerator", "/dev/stdin",
		)
		scanCmd.Stdin = pipeReader

		ignore := func(line string) {}

		if err := n.executeFn(scanCmd, ignore); err != nil {
			slog.Error("failed to scan repository", "error", err)
			return
		}
	}()

	return pipeWriter
}

type NoseyParkerScanner struct {
	*noseyParker
	*chain.Base
	npStdin        *io.PipeWriter
	continuePiping bool
	done           chan bool
}

func NewNoseyParkerScanner(configs ...cfg.Config) chain.Link {
	np := &NoseyParkerScanner{done: make(chan bool)}
	np.Base = chain.NewBase(np, configs...)
	return np
}

func (n *NoseyParkerScanner) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("datastore", "NoseyParker datastore file").WithDefault("datastore.np"),
		cfg.NewParam[bool]("continue_piping", "If true, pipes output to next link. If false, saves to datastore file.").WithDefault(true),
	}
}

func (n *NoseyParkerScanner) Initialize() error {
	n.noseyParker = newNoseyParker(n.done, n.ExecuteCmd)

	var err error
	n.datastore, err = cfg.As[string](n.Arg("datastore"))
	if err != nil {
		return fmt.Errorf("failed to get datastore: %w", err)
	}

	n.continuePiping, err = cfg.As[bool](n.Arg("continue_piping"))
	if err != nil {
		return fmt.Errorf("failed to get continue_piping: %w", err)
	}

	n.npStdin = n.startNpScanner()
	return nil
}

func (n *NoseyParkerScanner) Process(resource types.NPInput) error {
	encoder := json.NewEncoder(n.npStdin)
	if err := encoder.Encode(resource); err != nil {
		slog.Error("failed to encode data", "error", err)
		return fmt.Errorf("NoseyParkerScanner failed to encode data: %w", err)
	}

	return nil
}

func (n *NoseyParkerScanner) Complete() error {
	n.npStdin.Close()
	<-n.done

	if n.continuePiping {
		return n.pipeToNextLink()
	}

	return nil
}

func (n *NoseyParkerScanner) pipeToNextLink() error {
	reportCmd := exec.Command(
		"noseyparker",
		"report",
		"--datastore", n.datastore,
		"--format", "jsonl",
	)

	parser := func(line string) {
		if strings.TrimSpace(line) == "" {
			return
		}

		var output *types.NPOutput
		if err := json.Unmarshal([]byte(line), &output); err != nil {
			slog.Error("failed to parse finding", "error", err, "line", line)
			return
		}

		findings := output.ToNPFindings()
		for _, finding := range findings {
			n.Send(&finding)
		}
	}

	if err := n.ExecuteCmd(reportCmd, parser); err != nil {
		slog.Error("failed to report repository", "error", err)
		return err
	}

	return os.RemoveAll(n.datastore)
}

type NoseyParkerSummarizer struct {
	*chain.Base
	datastore string
}

func NewNoseyParkerSummarizer(configs ...cfg.Config) chain.Link {
	np := &NoseyParkerSummarizer{}
	np.Base = chain.NewBase(np, configs...)
	return np
}

func (n *NoseyParkerSummarizer) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("datastore", "datastore.np").WithDefault("datastore.np"),
	}
}

func (n *NoseyParkerSummarizer) Initialize() error {
	var err error
	n.datastore, err = cfg.As[string](n.Arg("datastore"))
	if err != nil {
		return fmt.Errorf("failed to get datastore: %w", err)
	}

	return nil
}

func (n *NoseyParkerSummarizer) Process(resource any) error {
	return nil
}

func (n *NoseyParkerSummarizer) Complete() error {
	summarizeCmd := exec.Command(
		"noseyparker",
		"summarize",
		"--datastore", n.datastore,
	)

	parser := func(line string) {
		n.Send(line)
	}

	if err := n.ExecuteCmd(summarizeCmd, parser); err != nil {
		slog.Error("failed to summarize repository", "error", err)
		return err
	}

	return nil
}

type NoseyParkerReporter struct {
	*chain.Base
	datastore string
}

func NewNoseyParkerReporter(configs ...cfg.Config) chain.Link {
	np := &NoseyParkerReporter{}
	np.Base = chain.NewBase(np, configs...)
	return np
}

func (n *NoseyParkerReporter) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("datastore", "datastore.np").WithDefault("datastore.np"),
	}
}

func (n *NoseyParkerReporter) Initialize() error {
	var err error
	n.datastore, err = cfg.As[string](n.Arg("datastore"))
	if err != nil {
		return fmt.Errorf("failed to get datastore: %w", err)
	}

	return nil
}

func (n *NoseyParkerReporter) Process(resource any) error {
	return nil
}

func (n *NoseyParkerReporter) Complete() error {
	summarizeCmd := exec.Command(
		"noseyparker",
		"report",
		"--datastore", n.datastore,
		"--format", "jsonl",
	)

	parser := func(line string) {
		var output *types.NPOutput
		if err := json.Unmarshal([]byte(line), &output); err != nil {
			slog.Error("failed to parse finding", "error", err, "line", line)
			return
		}

		findings := output.ToNPFindings()
		for _, finding := range findings {
			n.Send(&finding)
		}
	}

	if err := n.ExecuteCmd(summarizeCmd, parser); err != nil {
		slog.Error("failed to summarize repository", "error", err)
		return err
	}

	return nil
}
