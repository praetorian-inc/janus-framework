package cfg_test

import (
	"os/exec"
	"testing"

	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/stretchr/testify/assert"
)

func TestMethods(t *testing.T) {
	methods := cfg.NewMethodsHolder()

	didRun := false
	cmd := exec.Command("echo", "test")
	err := methods.ExecuteCmd(cmd, func(line string) {
		assert.Equal(t, line, "test")
		didRun = true
	})

	assert.NoError(t, err)
	assert.True(t, didRun, "method did not run")
}
