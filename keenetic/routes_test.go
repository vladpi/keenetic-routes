package keenetic

import (
	"testing"

	"github.com/vladpi/keenetic-routes/routes"
)

func TestBuildRouteIPv6CIDR(t *testing.T) {
	route, err := buildRoute(routes.Route{
		Host:      "2001:db8::/48",
		Interface: "Wireguard1",
		Auto:      true,
	})
	if err != nil {
		t.Fatalf("buildRoute: %v", err)
	}
	if route.Network == nil || route.Network.String() != "2001:db8::" {
		t.Fatalf("network: got %v", route.Network)
	}
	if route.PrefixLen == nil || int(*route.PrefixLen) != 48 {
		t.Fatalf("prefixlen: got %v", route.PrefixLen)
	}
	if route.Mask != nil {
		t.Fatalf("mask: expected nil, got %v", route.Mask)
	}
	if route.Interface == nil || route.Interface.String() != "Wireguard1" {
		t.Fatalf("interface: got %v", route.Interface)
	}
}
