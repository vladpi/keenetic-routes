package routes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "ip", input: "8.8.8.8", want: "8.8.8.8"},
		{name: "ip_trimmed", input: "  1.1.1.1  ", want: "1.1.1.1"},
		{name: "cidr", input: "192.168.0.0/16", want: "192.168.0.0/16"},
		{name: "ipv6", input: "2001:db8::1", wantErr: true},
		{name: "ipv6_cidr", input: "2001:db8::/32", wantErr: true},
		{name: "empty", input: "", wantErr: true},
		{name: "invalid_ip", input: "not-an-ip", wantErr: true},
		{name: "invalid_cidr", input: "10.0.0.0/33", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeHost(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadYAML_MissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.yaml")
	rf, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}
	if rf == nil {
		t.Fatalf("expected RoutesFile, got nil")
	}
	if len(rf.Routes) != 0 {
		t.Fatalf("expected empty RoutesFile, got %d routes", len(rf.Routes))
	}
}

func TestLoadYAML_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "routes.yaml")
	rf := &RoutesFile{
		Routes: []RouteGroup{
			{
				Comment: "test",
				Gateway: "192.168.1.1",
				Auto:    true,
				Hosts:   []string{"8.8.8.8"},
			},
		},
	}
	if err := SaveYAML(path, rf); err != nil {
		t.Fatalf("SaveYAML: %v", err)
	}
	loaded, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}
	if len(loaded.Routes) != 1 || len(loaded.Routes[0].Hosts) != 1 {
		t.Fatalf("unexpected loaded routes: %+v", loaded)
	}
}

func TestSaveYAML_CreatesDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "routes.yaml")
	rf := &RoutesFile{Routes: []RouteGroup{}}
	if err := SaveYAML(path, rf); err != nil {
		t.Fatalf("SaveYAML: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat saved file: %v", err)
	}
}
