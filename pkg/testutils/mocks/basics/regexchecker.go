package basics

import (
	"regexp"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

type RegexChecker struct {
	*chain.Base
	regex *regexp.Regexp
}

func NewRegexChecker(regex *regexp.Regexp, configs ...cfg.Config) chain.Link {
	rc := &RegexChecker{
		regex: regex,
	}
	rc.Base = chain.NewBase(rc, configs...)
	return rc
}

func (rc *RegexChecker) Params() []cfg.Param {
	return []cfg.Param{
		cfg.NewParam[string]("argument", "argument to be validated by regex").WithRegex(rc.regex),
	}
}

func (rc *RegexChecker) Process(_ string) error {
	return nil
}
