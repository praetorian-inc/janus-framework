package links

import (
	"log/slog"

	"github.com/praetorian-inc/janus-framework/pkg/chain"
	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
	"github.com/praetorian-inc/janus-framework/pkg/types"
	"github.com/praetorian-inc/janus-framework/pkg/util"
)

type Resolve struct {
	*chain.Base
}

func NewResolve(configs ...cfg.Config) chain.Link {
	r := &Resolve{}
	r.Base = chain.NewBase(r, configs...)
	return r
}

func (r *Resolve) Process(dw types.DomainWrapper) error {
	domain := dw.Domain
	ips := util.Resolve(domain)
	if len(ips) == 0 {
		slog.Debug("no ips found for", "domain", domain)
	}
	for _, ip := range ips {
		slog.Debug("found ip", "ip", ip, "domain", domain)
		r.Send(types.NewScannableAsset(ip))
	}
	return nil
}
