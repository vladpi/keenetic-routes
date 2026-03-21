package routes

import "testing"

type stubRoute struct {
	host      string
	network   string
	ip        string
	mask      string
	prefix    int
	prefixLen int
	comment   string
	gateway   string
	iface     string
	auto      bool
	reject    bool
}

func (s stubRoute) HostValue() string      { return s.host }
func (s stubRoute) NetworkValue() string   { return s.network }
func (s stubRoute) IPValue() string        { return s.ip }
func (s stubRoute) MaskValue() string      { return s.mask }
func (s stubRoute) PrefixValue() int       { return s.prefix }
func (s stubRoute) PrefixLenValue() int    { return s.prefixLen }
func (s stubRoute) CommentValue() string   { return s.comment }
func (s stubRoute) GatewayValue() string   { return s.gateway }
func (s stubRoute) InterfaceValue() string { return s.iface }
func (s stubRoute) AutoValue() bool        { return s.auto }
func (s stubRoute) RejectValue() bool      { return s.reject }

func TestRouteDestAndToYAML(t *testing.T) {
	r1 := stubRoute{
		host:    "8.8.8.8",
		comment: "test",
		gateway: "10.0.0.1",
		auto:    true,
	}
	r2 := stubRoute{
		network: "192.168.0.0",
		mask:    "255.255.255.0",
		comment: "test",
		gateway: "10.0.0.1",
		auto:    true,
	}
	r3 := stubRoute{
		host:    "2001:db8::1",
		comment: "ipv6",
	}
	r4 := stubRoute{
		network:   "2001:db8::",
		prefixLen: 32,
		comment:   "ipv6-cidr",
	}

	if got := RouteDest(r1); got != "8.8.8.8" {
		t.Fatalf("RouteDest host: got %q", got)
	}
	if got := RouteDest(r2); got != "192.168.0.0/24" {
		t.Fatalf("RouteDest network: got %q", got)
	}
	if got := RouteDest(r3); got != "2001:db8::1" {
		t.Fatalf("RouteDest ipv6: got %q", got)
	}
	if got := RouteDest(r4); got != "2001:db8::/32" {
		t.Fatalf("RouteDest ipv6 cidr: got %q", got)
	}

	rf := ToYAML([]Route{
		{Host: "8.8.8.8", Comment: "test", Gateway: "10.0.0.1", Auto: true},
		{Host: "192.168.0.0/24", Comment: "test", Gateway: "10.0.0.1", Auto: true},
		{Host: "2001:db8::/32", Comment: "test", Gateway: "10.0.0.1", Auto: true},
	})
	if rf == nil || len(rf.Routes) != 1 {
		t.Fatalf("expected 1 group, got %+v", rf)
	}
	group := rf.Routes[0]
	if group.Comment != "test" || group.Gateway != "10.0.0.1" || !group.Auto {
		t.Fatalf("unexpected group metadata: %+v", group)
	}
	if len(group.Hosts) != 3 {
		t.Fatalf("expected 3 hosts, got %d", len(group.Hosts))
	}
}
