package util

import (
	"net"
	"regexp"
)

const (
	AssetTypeDomain string = "domain"
	AssetTypeIPv4   string = "ipv4"
	AssetTypeIPv6   string = "ipv6"
	AssetTypeCIDR   string = "cidr"

	NoType string = ""
)

var (
	domain = regexp.MustCompile(`^(https?://)?((xn--[a-zA-Z0-9-]+|[a-zA-Z0-9-]+)\.)+([a-zA-Z]{2,})$`)
)

func TypeFromTarget(target string) string {
	_, _, err := net.ParseCIDR(target)
	if err == nil {
		return AssetTypeCIDR
	}

	ipType := IPType(target)
	if ipType == "ipv4" {
		return AssetTypeIPv4
	}
	if ipType == "ipv6" {
		return AssetTypeIPv6
	}
	if domain.MatchString(target) {
		return AssetTypeDomain
	}
	return NoType
}

func IPType(s string) string {
	ip := net.ParseIP(s)
	if ip == nil {
		return ""
	}
	if ip4 := ip.To4(); ip4 != nil {
		return "ipv4"
	}
	_, _, err := net.ParseCIDR(s)
	if err == nil {
		return "cidr"
	}
	return "ipv6"
}
