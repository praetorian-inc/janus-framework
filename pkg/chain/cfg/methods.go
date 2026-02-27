package cfg

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
)

type InjectableMethods interface {
	ExecuteCmd(cmd *exec.Cmd, lineparser func(string)) error
	ExecuteCmdAll(cmd *exec.Cmd) ([]byte, error)
}

type MethodsHolder struct {
	InjectableMethods
}

func NewMethodsHolder() *MethodsHolder {
	return &MethodsHolder{
		InjectableMethods: &MethodsImpl{},
	}
}

func (m *MethodsHolder) SetMethods(methods InjectableMethods) {
	m.InjectableMethods = methods
}

type MethodsImpl struct {
}

func (m *MethodsImpl) ExecuteCmd(cmd *exec.Cmd, lineparser func(string)) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("could not get stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("could not get stderr pipe: %v", err)
	}
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start: %q: %s", strings.Join(cmd.Args, " "), err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	processOutput := func(input io.Reader, buffer *bytes.Buffer) {
		defer wg.Done()
		reader := bufio.NewReaderSize(input, 4*1024*1024) // 4MB line limit
		for {
			line, isPrefix, err := reader.ReadLine()
			if err != nil {
				if err != io.EOF {
					slog.Error("failed to read output", "error", err)
				}
				return
			}

			if isPrefix {
				// Line exceeded 4MB, drain the rest and skip it
				slog.Warn("skipping oversized output line")
				for isPrefix {
					_, isPrefix, err = reader.ReadLine()
					if err != nil {
						return
					}
				}
				continue
			}

			text := string(line)
			buffer.WriteString(text)
			lineparser(text)
		}
	}

	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}

	go processOutput(stdout, stdoutBuffer)
	go processOutput(stderr, stderrBuffer)

	wg.Wait()

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("%s", stderrBuffer.String())
	}

	return nil
}

func (m *MethodsImpl) ExecuteCmdAll(cmd *exec.Cmd) ([]byte, error) {
	output, err := cmd.Output()
	if err != nil {
		stderr := err.Error()

		var errOutput *exec.ExitError
		if errors.As(err, &errOutput) {
			stderr = string(errOutput.Stderr)
		}

		return nil, fmt.Errorf("failed to execute: %q: %s", strings.Join(cmd.Args, " "), stderr)
	}
	return output, nil
}
