package routes

// routeGroupKey identifies a unique group by its shared route parameters.
type routeGroupKey struct {
	comment string
	gateway string
	iface   string
	auto    bool
	reject  bool
}

// ToYAML builds a RoutesFile from domain routes, grouping by comment and params.
func ToYAML(routesList []Route) *RoutesFile {
	grouped := make(map[routeGroupKey][]string)
	var order []routeGroupKey

	for _, r := range routesList {
		if r.Host == "" || !isIPv4OrCIDR(r.Host) {
			continue
		}
		k := routeGroupKey{
			comment: r.Comment,
			gateway: r.Gateway,
			iface:   r.Interface,
			auto:    r.Auto,
			reject:  r.Reject,
		}
		if _, exists := grouped[k]; !exists {
			order = append(order, k)
		}
		grouped[k] = append(grouped[k], r.Host)
	}

	groups := make([]RouteGroup, 0, len(order))
	for _, k := range order {
		groups = append(groups, RouteGroup{
			Comment:   k.comment,
			Gateway:   k.gateway,
			Interface: k.iface,
			Auto:      k.auto,
			Reject:    k.reject,
			Hosts:     grouped[k],
		})
	}
	return &RoutesFile{Routes: groups}
}
