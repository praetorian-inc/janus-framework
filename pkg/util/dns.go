package util

import (
	"context"
	"net"
)

func Resolve(host string) []string {
	resolver := &net.Resolver{}

	hits, err := resolver.LookupIPAddr(context.Background(), host)
	if err != nil {
		return []string{}
	}

	var addresses []string
	for _, ip := range hits {
		if ip.IP.IsPrivate() || ip.IP.IsLoopback() {
			continue
		}
		addresses = append(addresses, ip.IP.String())
	}
	return addresses
}
