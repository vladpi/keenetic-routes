package routes

import (
	"fmt"
	"net"
	"strings"
)

// RouteView is a minimal interface for converting external routes.
type RouteView interface {
	HostValue() string
	NetworkValue() string
	IPValue() string
	MaskValue() string
	PrefixValue() int
	PrefixLenValue() int
	CommentValue() string
	GatewayValue() string
	InterfaceValue() string
	AutoValue() bool
	RejectValue() bool
}

// RouteDest extracts destination from a route: "host" (IPv4 or CIDR), or "network"/"ip" + "mask"/"prefix".
// Returns empty string if the destination is missing or not IPv4.
func RouteDest(r RouteView) string {
	if h := r.HostValue(); h != "" {
		if !isIPv4OrCIDR(h) {
			return ""
		}
		return h
	}
	network := r.NetworkValue()
	if network == "" {
		network = r.IPValue()
	}
	if network == "" {
		return ""
	}
	if net.ParseIP(network).To4() == nil {
		return ""
	}
	if prefix := r.PrefixValue(); prefix > 0 && prefix <= 32 {
		return fmt.Sprintf("%s/%d", network, prefix)
	}
	if prefix := r.PrefixLenValue(); prefix > 0 && prefix <= 32 {
		return fmt.Sprintf("%s/%d", network, prefix)
	}
	maskStr := r.MaskValue()
	if maskStr == "" {
		return network
	}
	maskIP := net.ParseIP(maskStr)
	if maskIP == nil {
		return network
	}
	if maskIP = maskIP.To4(); maskIP == nil {
		return network
	}
	ones, _ := net.IPv4Mask(maskIP[0], maskIP[1], maskIP[2], maskIP[3]).Size()
	return fmt.Sprintf("%s/%d", network, ones)
}

func isIPv4OrCIDR(s string) bool {
	if strings.Contains(s, "/") {
		ip, _, err := net.ParseCIDR(s)
		if err != nil {
			return false
		}
		return ip.To4() != nil
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return false
	}
	return ip.To4() != nil
}
