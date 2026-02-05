package keenetic

// NDMS RCI API documentation (routes, auth, field names):
//   https://help.keenetic.com — раздел «Удалённое управление» / NDMS RCI API
//   Поиск по сайту: "NDMS RCI" или "rci/ip/route"

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"

	"github.com/vladpi/keenetic-routes/routes"
)

const routeBatchSize = 50

type Stringish string

func (s *Stringish) UnmarshalJSON(data []byte) error {
	if s == nil {
		return nil
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	if v == nil {
		*s = ""
		return nil
	}
	*s = Stringish(strings.TrimSpace(fmt.Sprint(v)))
	return nil
}

func (s Stringish) String() string {
	return string(s)
}

type Boolish bool

func (b *Boolish) UnmarshalJSON(data []byte) error {
	if b == nil {
		return nil
	}
	s := strings.TrimSpace(string(data))
	if s == "" || s == "null" {
		*b = false
		return nil
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = strings.TrimSpace(strings.Trim(s, `"`))
	}
	switch strings.ToLower(s) {
	case "true", "1", "yes":
		*b = true
		return nil
	case "false", "0", "no":
		*b = false
		return nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		*b = f != 0
		return nil
	}
	*b = false
	return nil
}

type Intish int

func (i *Intish) UnmarshalJSON(data []byte) error {
	if i == nil {
		return nil
	}
	s := strings.TrimSpace(string(data))
	if s == "" || s == "null" {
		*i = 0
		return nil
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = strings.TrimSpace(strings.Trim(s, `"`))
	}
	if n, err := strconv.Atoi(s); err == nil {
		*i = Intish(n)
		return nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && f == math.Trunc(f) {
		*i = Intish(int(f))
		return nil
	}
	*i = 0
	return nil
}

type Route struct {
	Host      *Stringish `json:"host,omitempty"`
	Network   *Stringish `json:"network,omitempty"`
	IP        *Stringish `json:"ip,omitempty"`
	Mask      *Stringish `json:"mask,omitempty"`
	Prefix    *Intish    `json:"prefix,omitempty"`
	PrefixLen *Intish    `json:"prefixlen,omitempty"`
	Comment   *Stringish `json:"comment,omitempty"`
	Gateway   *Stringish `json:"gateway,omitempty"`
	Interface *Stringish `json:"interface,omitempty"`
	Auto      *Boolish   `json:"auto,omitempty"`
	Reject    *Boolish   `json:"reject,omitempty"`
	No        *bool      `json:"no,omitempty"`
}

func (r Route) HostValue() string {
	return stringValue(r.Host)
}

func (r Route) NetworkValue() string {
	return stringValue(r.Network)
}

func (r Route) IPValue() string {
	return stringValue(r.IP)
}

func (r Route) MaskValue() string {
	return stringValue(r.Mask)
}

func (r Route) PrefixValue() int {
	return intValue(r.Prefix)
}

func (r Route) PrefixLenValue() int {
	return intValue(r.PrefixLen)
}

func (r Route) CommentValue() string {
	return stringValue(r.Comment)
}

func (r Route) GatewayValue() string {
	return stringValue(r.Gateway)
}

func (r Route) InterfaceValue() string {
	return stringValue(r.Interface)
}

func (r Route) AutoValue() bool {
	return boolValue(r.Auto)
}

func (r Route) RejectValue() bool {
	return boolValue(r.Reject)
}

type RouteEnvelope struct {
	IP RouteWrapper `json:"ip"`
}

type RouteWrapper struct {
	Route Route `json:"route"`
}

type SaveConfig struct {
	System SystemConfig `json:"system"`
}

type SystemConfig struct {
	Configuration ConfigSave `json:"configuration"`
}

type ConfigSave struct {
	Save bool `json:"save"`
}

func routeEnvelope(route Route) RouteEnvelope {
	return RouteEnvelope{IP: RouteWrapper{Route: route}}
}

func saveConfigPayload() SaveConfig {
	return SaveConfig{System: SystemConfig{Configuration: ConfigSave{Save: true}}}
}

func stringishPtr(v string) *Stringish {
	s := Stringish(v)
	return &s
}

func boolishPtr(v bool) *Boolish {
	b := Boolish(v)
	return &b
}

func boolPtr(v bool) *bool {
	return &v
}

func stringValue(v *Stringish) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(v.String())
}

func boolValue(v *Boolish) bool {
	if v == nil {
		return false
	}
	return bool(*v)
}

func intValue(v *Intish) int {
	if v == nil {
		return 0
	}
	return int(*v)
}

func toDomainRoutes(raw []Route) ([]routes.Route, error) {
	out := make([]routes.Route, 0, len(raw))
	for _, r := range raw {
		dest := routes.RouteDest(r)
		if dest == "" {
			return nil, fmt.Errorf("unsupported route destination (IPv4 only): host=%q network=%q ip=%q", r.HostValue(), r.NetworkValue(), r.IPValue())
		}
		out = append(out, routes.Route{
			Host:      dest,
			Comment:   r.CommentValue(),
			Gateway:   r.GatewayValue(),
			Interface: r.InterfaceValue(),
			Auto:      r.AutoValue(),
			Reject:    r.RejectValue(),
		})
	}
	return out, nil
}

// GetRoutes returns current static routes from the router (GET rci/ip/route).
func (c *Client) GetRoutes() ([]Route, error) {
	data, err := c.Request("rci/ip/route", nil)
	if err != nil {
		return nil, err
	}
	var routes []Route
	if err := json.Unmarshal(data, &routes); err != nil {
		return nil, fmt.Errorf("decode routes: %w", err)
	}
	return routes, nil
}

// GetDomainRoutes returns current static routes converted to the domain model.
func (c *Client) GetDomainRoutes() ([]routes.Route, error) {
	raw, err := c.GetRoutes()
	if err != nil {
		return nil, err
	}
	return toDomainRoutes(raw)
}

// DeleteAllRoutes fetches current routes and sends delete (no: true) for each, then save.
func (c *Client) DeleteAllRoutes() error {
	routes, err := c.GetRoutes()
	if err != nil {
		return err
	}
	var payload []any
	for i := range routes {
		routes[i].No = boolPtr(true)
		payload = append(payload, routeEnvelope(routes[i]))
	}
	payload = append(payload, saveConfigPayload())
	_, err = c.Request("rci/", payload)
	return err
}

// AddRoutes adds static routes from entries (each with its own params), then save. Sends in batches.
func (c *Client) AddRoutes(entries []routes.Route) error {
	if len(entries) == 0 {
		return nil
	}
	for i := 0; i < len(entries); i += routeBatchSize {
		end := min(i+routeBatchSize, len(entries))
		batch := entries[i:end]
		var payload []any
		for _, e := range batch {
			route, err := buildRoute(e)
			if err != nil {
				return fmt.Errorf("add routes: %w", err)
			}
			payload = append(payload, routeEnvelope(route))
		}
		payload = append(payload, saveConfigPayload())
		if _, err := c.Request("rci/", payload); err != nil {
			return fmt.Errorf("add routes batch at %d: %w", i, err)
		}
	}
	return nil
}

func buildRoute(e routes.Route) (Route, error) {
	route := Route{
		Auto:    boolishPtr(e.Auto),
		Comment: stringishPtr(e.Comment),
	}
	if strings.Contains(e.Host, "/") {
		_, ipNet, err := net.ParseCIDR(e.Host)
		if err != nil {
			return Route{}, fmt.Errorf("invalid CIDR %q: %w", e.Host, err)
		}
		route.Network = stringishPtr(ipNet.IP.String())
		route.Mask = stringishPtr(net.IP(ipNet.Mask).String())
	} else {
		route.Host = stringishPtr(e.Host)
	}
	if e.Reject {
		route.Reject = boolishPtr(true)
	}
	if e.Gateway != "" {
		route.Gateway = stringishPtr(e.Gateway)
	}
	if e.Interface != "" {
		route.Interface = stringishPtr(e.Interface)
	}
	return route, nil
}
