package chain_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/testutils/mocks"
	"github.com/praetorian-inc/janus-framework/pkg/testutils/mocks/basics"
	"github.com/praetorian-inc/janus-framework/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLink(t *testing.T) {
	link := basics.NewStrLink()
	go func() {
		chain.Process(link, "123")
		link.Close()
	}()

	received := []string{}
	for output, ok := chain.RecvAs[string](link); ok; output, ok = chain.RecvAs[string](link) {
		received = append(received, output)
	}

	assert.Equal(t, []string{"123"}, received)
}

func TestNewLink_Params(t *testing.T) {
	link := basics.NewParamsLink()

	assert.Equal(t, 3, len(link.Params()))
	assert.True(t, link.HasParam("optional"))
	assert.True(t, link.HasParam("required"))
	assert.True(t, link.HasParam("default"))
}

func TestNewLink_Arg(t *testing.T) {
	didRun := false
	assertFunc := func(arg string, err error) {
		assert.NoError(t, err)
		assert.Equal(t, "123", arg)
		didRun = true
	}

	link := basics.NewArgCheckingLink(assertFunc, cfg.WithArg("argument", "123"))
	go func() {
		chain.Process(link, "123")
		link.Close()
	}()

	chain.RecvAs[string](link)
	assert.True(t, didRun, "link did not run")
}

func TestLink_Permissions(t *testing.T) {
	link := basics.NewPermissionsLink().
		WithPermissions(
			cfg.NewPermission(cfg.AWS, "permission1"),
			cfg.NewPermission(cfg.GCP, "permission2"),
			cfg.NewPermission(cfg.Azure, "permission3"),
		)

	actual := []string{}
	for _, p := range link.Permissions() {
		actual = append(actual, p.String())
	}

	assert.Equal(t, []string{"AWS:permission1", "GCP:permission2", "Azure:permission3"}, actual)
}

func TestLink_Invoke(t *testing.T) {
	link := basics.NewStrLink()

	results, err := link.Invoke("123", "456", "789")

	assert.NoError(t, err)
	assert.Equal(t, []any{"123", "456", "789"}, results)
}

func TestLink_InvokeNil(t *testing.T) {
	link := basics.NewStrLink()

	results, err := link.Invoke(nil)

	assert.NoError(t, err)
	assert.Equal(t, []any{}, results)
}

func TestLink_InvokeStruct(t *testing.T) {
	link := mocks.NewSubdomain()

	results, err := link.Invoke(types.DomainWrapper{Domain: "example.com"})

	expected := []string{"sub1.example.com", "sub2.example.com", "sub3.example.com"}

	assert.NoError(t, err)
	for i, result := range results {
		asset, ok := result.(*types.ScannableAsset)
		require.True(t, ok)
		assert.Equal(t, expected[i], asset.Domain)
	}
}

func TestLink_Logging(t *testing.T) {
	w := &bytes.Buffer{}

	link := basics.NewLoggingLink()
	loggingLink := link.(*basics.LoggingLink)
	loggingLink.Base.Logger.SetWriter(w) // ugly, but it's just one unit test

	link.Invoke(basics.Msg{Level: slog.LevelInfo, Message: "test"})

	assert.Contains(t, w.String(), "level=INFO link=*basics.LoggingLink msg=test")
}
