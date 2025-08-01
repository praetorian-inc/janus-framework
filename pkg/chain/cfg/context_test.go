package cfg_test

import (
	"context"
	"testing"

	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	holder := cfg.NewContextHolder()

	assert.Equal(t, holder.Context(), context.Background())

	newCtx := context.WithValue(context.Background(), "test", "expected")
	holder.SetContext(newCtx)

	assert.Equal(t, holder.Context(), newCtx)

	unexpectedCtx := context.WithValue(context.Background(), "test", "unexpected")
	assert.NotEqual(t, holder.Context(), unexpectedCtx)
}
