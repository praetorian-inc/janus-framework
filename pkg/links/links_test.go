package links_test

import (
	"os"
	"testing"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/links"
	"github.com/praetorian-inc/janus-framework/pkg/output"
	"github.com/praetorian-inc/janus-framework/pkg/testutils/mocks"
	"github.com/stretchr/testify/assert"
)

func TestLinks_Converter(t *testing.T) {
	c := chain.NewChain(
		links.NewStringConverter(mocks.NewSubdomain()),
		mocks.NewSubdomain(),
	).WithOutputters(
		output.NewJSONOutputter(),
	).WithConfigs(
		cfg.WithArg("jsonoutfile", "test.json"),
	)

	c.Send(`{"domain": "example.com"}`)
	c.Close()
	c.Wait()

	data, err := os.ReadFile("test.json")
	assert.NoError(t, err)
	assert.Equal(t, `[{"target":"sub1.example.com","type":"domain","domain":"sub1.example.com"},{"target":"sub2.example.com","type":"domain","domain":"sub2.example.com"},{"target":"sub3.example.com","type":"domain","domain":"sub3.example.com"}]`+"\n", string(data))

	os.Remove("test.json")
}
