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

// RouteDest extracts destination from a route: "host" (IP or CIDR), or "network"/"ip" + "mask"/"prefix".
// Returns empty string if the destination is missing or invalid.
func RouteDest(r RouteView) string {
	if h := r.HostValue(); h != "" {
		if !isIPOrCIDR(h) {
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
	networkIP := net.ParseIP(network)
	if networkIP == nil {
		return ""
	}
	network = networkIP.String()
	maxPrefix := 128
	if networkIP.To4() != nil {
		maxPrefix = 32
	}
	if prefix := r.PrefixValue(); prefix > 0 && prefix <= maxPrefix {
		return fmt.Sprintf("%s/%d", network, prefix)
	}
	if prefix := r.PrefixLenValue(); prefix > 0 && prefix <= maxPrefix {
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

func isIPOrCIDR(s string) bool {
	if strings.Contains(s, "/") {
		ip, _, err := net.ParseCIDR(s)
		if err != nil {
			return false
		}
		return ip != nil
	}
	ip := net.ParseIP(s)
	return ip != nil
}
