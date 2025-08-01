package noseyparker

import (
	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/links"
	"github.com/praetorian-inc/janus-framework/pkg/types"
)

func NewConvertToNPInput(configs ...cfg.Config) chain.Link {
	return links.FromTransformerSlice(func(input types.CanNPInput) ([]types.NPInput, error) {
		return input.ToNPInputs()
	})
}
