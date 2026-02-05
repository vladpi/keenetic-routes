package routes

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Route is a full route: host plus all Keenetic parameters.
type Route struct {
	Host      string
	Comment   string
	Gateway   string
	Interface string
	Auto      bool
	Reject    bool
}

// RouteGroup is a YAML group: shared params and list of hosts.
type RouteGroup struct {
	Comment   string   `yaml:"comment,omitempty"`
	Gateway   string   `yaml:"gateway,omitempty"`
	Interface string   `yaml:"interface,omitempty"`
	Auto      bool     `yaml:"auto,omitempty"`
	Reject    bool     `yaml:"reject,omitempty"`
	Hosts     []string `yaml:"hosts"`
}

// RoutesFile is the root YAML structure.
type RoutesFile struct {
	Routes []RouteGroup `yaml:"routes"`
}

// normalizeHost validates and normalizes "IP" or "IP/CIDR" to a string form suitable for Keenetic host field.
func normalizeHost(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("empty host")
	}
	if strings.Contains(s, "/") {
		ip, n, err := net.ParseCIDR(s)
		if err != nil {
			return "", err
		}
		if ip.To4() == nil {
			return "", fmt.Errorf("invalid IPv4 CIDR")
		}
		return n.String(), nil
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return "", fmt.Errorf("invalid IP")
	}
	if ip.To4() == nil {
		return "", fmt.Errorf("invalid IPv4")
	}
	return ip.String(), nil
}

// LoadYAML reads a YAML routes file. Returns nil RoutesFile and nil error if file does not exist (for merge).
func LoadYAML(path string) (*RoutesFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &RoutesFile{Routes: nil}, nil
		}
		return nil, fmt.Errorf("read file: %w", err)
	}
	var rf RoutesFile
	if err := yaml.Unmarshal(data, &rf); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}
	if rf.Routes == nil {
		rf.Routes = []RouteGroup{}
	}
	return &rf, nil
}

// SaveYAML writes RoutesFile to path as YAML.
func SaveYAML(path string, rf *RoutesFile) error {
	if rf == nil {
		rf = &RoutesFile{Routes: []RouteGroup{}}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	data, err := yaml.Marshal(rf)
	if err != nil {
		return fmt.Errorf("marshal YAML: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// FlattenToEntries converts RoutesFile to a slice of Route (one per host), normalizing hosts.
func FlattenToEntries(rf *RoutesFile) ([]Route, error) {
	if rf == nil || len(rf.Routes) == 0 {
		return nil, nil
	}
	var out []Route
	for _, g := range rf.Routes {
		if len(g.Hosts) == 0 {
			continue
		}
		hasGW := g.Gateway != ""
		hasIface := g.Interface != ""
		if hasGW == hasIface {
			return nil, fmt.Errorf("group %q: set exactly one of gateway or interface", g.Comment)
		}
		for _, h := range g.Hosts {
			norm, err := normalizeHost(h)
			if err != nil {
				return nil, fmt.Errorf("group %q host %q: %w", g.Comment, h, err)
			}
			out = append(out, Route{
				Host:      norm,
				Comment:   g.Comment,
				Gateway:   g.Gateway,
				Interface: g.Interface,
				Auto:      g.Auto,
				Reject:    g.Reject,
			})
		}
	}
	return out, nil
}
