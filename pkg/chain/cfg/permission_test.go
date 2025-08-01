package cfg_test

import (
	"testing"

	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/stretchr/testify/assert"
)

func TestPermission_String(t *testing.T) {
	s3Read := cfg.NewPermission(cfg.AWS, "s3:GetObject")
	githubRead := cfg.NewPermission(cfg.GitHub, "Repository:Contents")

	assert.Equal(t, "AWS:s3:GetObject", s3Read.String())
	assert.Equal(t, "GitHub:Repository:Contents", githubRead.String())
}
