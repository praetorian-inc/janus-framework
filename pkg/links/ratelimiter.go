package links

import (
	"log/slog"
	"slices"
	"time"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cherrors"
)

type RateLimiter struct {
	*chain.Base
	index int
}

func NewRateLimiter(configs ...cfg.Config) chain.Link {
	r := &RateLimiter{}
	r.Base = chain.NewBase(r, configs...)
	return r
}

func (r *RateLimiter) Process(input string) error {
	rateLimitOn, err := cfg.As[[]int](r.Arg("rateLimitOn"))
	if err != nil {
		return err
	}

	r.index++
	if slices.Contains(rateLimitOn, r.index-1) {
		return cherrors.ErrRateLimited
	}

	time.Sleep(1 * time.Second)
	r.Send(input)
	return nil
}

func (r *RateLimiter) HandleRateLimited() {
	slog.Warn("rate limited exceeded")
	time.Sleep(5 * time.Second)
}

func (r *RateLimiter) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[[]int]("rateLimitOn", "indexes to rate limit on").WithDefault([]int{}),
	}
}
