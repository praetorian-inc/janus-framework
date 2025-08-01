package types

import (
	"github.com/praetorian-inc/janus/pkg/util"
)

type ScannableAsset struct {
	Target string `json:"target"`
	Type   string `json:"type"`
	Domain string `json:"domain,omitempty"`
	IP     string `json:"ip,omitempty"`
	CIDR   string `json:"cidr,omitempty"`
	Ports  []Port `json:"ports,omitempty"`
}

func (sa *ScannableAsset) HasPort(transport Transport, port string) bool {
	for _, p := range sa.Ports {
		if p.Port == port && p.Transport == transport {
			return true
		}
	}
	return false
}

func (sa *ScannableAsset) AddPort(transport Transport, port string) {
	if sa.HasPort(transport, port) {
		return
	}
	sa.Ports = append(sa.Ports, Port{Transport: transport, Port: port})
}

func NewScannableAsset(target string) *ScannableAsset {
	sa := &ScannableAsset{
		Target: target,
		Type:   util.TypeFromTarget(target),
	}

	switch sa.Type {
	case util.AssetTypeDomain:
		sa.Domain = target
	case util.AssetTypeIPv4, util.AssetTypeIPv6:
		sa.IP = target
	case util.AssetTypeCIDR:
		sa.CIDR = target
	}

	return sa
}
