package testutils

import "os/exec"

type MockMethods struct {
	ExecuteFn    func(cmd *exec.Cmd, lineparser func(string)) error
	ExecuteAllFn func(cmd *exec.Cmd) ([]byte, error)
}

func (m *MockMethods) ExecuteCmd(cmd *exec.Cmd, lineparser func(string)) error {
	m.ExecuteFn(cmd, lineparser)
	return nil
}

func (m *MockMethods) ExecuteCmdAll(cmd *exec.Cmd) ([]byte, error) {
	return m.ExecuteAllFn(cmd)
}
