package routes

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

const domainLookupTimeout = 5 * time.Second

// ResolveSummary describes the result of domain resolution.
type ResolveSummary struct {
	Groups   int
	Domains  int
	IPsAdded int
}

// IPResolver is a minimal DNS resolver interface.
type IPResolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// ResolveDomains resolves RouteGroup.Domains and merges IPv4 results into Hosts.
func ResolveDomains(rf *RoutesFile) (ResolveSummary, error) {
	return ResolveDomainsWithResolver(rf, net.DefaultResolver)
}

// ResolveDomainsWithResolver resolves domains using the provided resolver.
func ResolveDomainsWithResolver(rf *RoutesFile, resolver IPResolver) (ResolveSummary, error) {
	var summary ResolveSummary
	if rf == nil || len(rf.Routes) == 0 {
		return summary, nil
	}
	for i := range rf.Routes {
		group := &rf.Routes[i]
		if len(group.Domains) == 0 {
			continue
		}
		if (group.Gateway == "") == (group.Interface == "") {
			return summary, fmt.Errorf("group %s: set exactly one of gateway or interface", groupLabel(group, i))
		}
		summary.Groups++

		seenHosts := make(map[string]struct{})
		mergedHosts := make([]string, 0, len(group.Hosts))
		for _, h := range group.Hosts {
			trimmed := strings.TrimSpace(h)
			if trimmed == "" {
				continue
			}
			if _, exists := seenHosts[trimmed]; exists {
				continue
			}
			seenHosts[trimmed] = struct{}{}
			mergedHosts = append(mergedHosts, trimmed)
		}

		seenDomains := make(map[string]struct{})
		for _, d := range group.Domains {
			domain := strings.TrimSpace(d)
			if domain == "" {
				return summary, fmt.Errorf("group %s: empty domain entry", groupLabel(group, i))
			}
			if _, exists := seenDomains[domain]; exists {
				continue
			}
			seenDomains[domain] = struct{}{}
			summary.Domains++

			ips, err := lookupIPv4(resolver, domain)
			if err != nil {
				return summary, fmt.Errorf("group %s domain %q: %w", groupLabel(group, i), domain, err)
			}
			if len(ips) == 0 {
				return summary, fmt.Errorf("group %s domain %q: no IPv4 records found", groupLabel(group, i), domain)
			}
			for _, ip := range ips {
				if _, exists := seenHosts[ip]; exists {
					continue
				}
				seenHosts[ip] = struct{}{}
				mergedHosts = append(mergedHosts, ip)
				summary.IPsAdded++
			}
		}

		group.Hosts = mergedHosts
	}
	return summary, nil
}

func lookupIPv4(resolver IPResolver, domain string) ([]string, error) {
	if ip := net.ParseIP(domain); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			return []string{ip4.String()}, nil
		}
		return nil, fmt.Errorf("IPv6 is not supported")
	}
	ctx, cancel := context.WithTimeout(context.Background(), domainLookupTimeout)
	defer cancel()

	addrs, err := resolver.LookupIPAddr(ctx, domain)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	var ips []string
	for _, addr := range addrs {
		if ip4 := addr.IP.To4(); ip4 != nil {
			s := ip4.String()
			if _, exists := seen[s]; exists {
				continue
			}
			seen[s] = struct{}{}
			ips = append(ips, s)
		}
	}
	return ips, nil
}

func groupLabel(group *RouteGroup, idx int) string {
	if group != nil && group.Comment != "" {
		return fmt.Sprintf("%q", group.Comment)
	}
	return fmt.Sprintf("#%d", idx+1)
}
