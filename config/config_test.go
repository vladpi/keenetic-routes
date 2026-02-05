package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func withTempHome(t *testing.T, fn func(dir string)) {
	t.Helper()
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})
	t.Setenv("HOME", dir)
	fn(dir)
}

func TestLoadConfig_PriorityAndMerging(t *testing.T) {
	tests := []struct {
		name         string
		configYAML   string
		env          map[string]string
		envFile      string
		hostFlag     string
		userFlag     string
		passwordFlag string
		want         Config
	}{
		{
			name:       "config_over_env",
			configYAML: "host: 10.0.0.1:280\nuser: cfg\npassword: cfgpass\n",
			env: map[string]string{
				"KEENETIC_HOST":     "10.0.0.2:280",
				"KEENETIC_USER":     "env",
				"KEENETIC_PASSWORD": "envpass",
			},
			want: Config{Host: "10.0.0.1:280", User: "cfg", Password: "cfgpass"},
		},
		{
			name:       "env_fills_missing",
			configYAML: "host: 10.0.0.3:280\n",
			env: map[string]string{
				"KEENETIC_USER":     "envuser",
				"KEENETIC_PASSWORD": "envpass",
			},
			want: Config{Host: "10.0.0.3:280", User: "envuser", Password: "envpass"},
		},
		{
			name:         "flags_override_partial",
			configYAML:   "host: 10.0.0.4:280\nuser: cfguser\npassword: cfgpass\n",
			hostFlag:     "10.0.0.9:280",
			userFlag:     "",
			passwordFlag: "",
			want:         Config{Host: "10.0.0.9:280", User: "cfguser", Password: "cfgpass"},
		},
		{
			name:    "dotenv_lowest_priority",
			envFile: "KEENETIC_HOST=10.0.0.5:280\nKEENETIC_USER=dotenv\nKEENETIC_PASSWORD=dotenvpass\n",
			want:    Config{Host: "10.0.0.5:280", User: "dotenv", Password: "dotenvpass"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTempHome(t, func(dir string) {
				if tt.configYAML != "" {
					configPath := filepath.Join(dir, ".config", "keenetic-routes", "config.yaml")
					writeFile(t, configPath, tt.configYAML)
				}
				if tt.envFile != "" {
					writeFile(t, filepath.Join(dir, ".env"), tt.envFile)
				}
				for k, v := range tt.env {
					t.Setenv(k, v)
				}

				cfg, err := LoadConfig(tt.hostFlag, tt.userFlag, tt.passwordFlag)
				if err != nil {
					t.Fatalf("LoadConfig: %v", err)
				}
				if cfg.Host != tt.want.Host || cfg.User != tt.want.User || cfg.Password != tt.want.Password {
					t.Fatalf("got %+v, want %+v", *cfg, tt.want)
				}
			})
		})
	}
}
