package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config holds the Keenetic router connection configuration.
type Config struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

// LoadConfig loads configuration from multiple sources in priority order:
// 1. Command line flags (passed as parameters)
// 2. Config file (~/.config/keenetic-routes/config.yaml)
// 3. Environment variables
// 4. .env file in current directory
func LoadConfig(hostFlag, userFlag, passwordFlag string) (*Config, error) {
	cfg := &Config{}

	configFile := getConfigFilePath()
	if data, err := os.ReadFile(configFile); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config file %s: %w", configFile, err)
		}
	}

	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	if cfg.Host == "" {
		cfg.Host = os.Getenv("KEENETIC_HOST")
	}
	if cfg.User == "" {
		cfg.User = os.Getenv("KEENETIC_USER")
	}
	if cfg.Password == "" {
		cfg.Password = os.Getenv("KEENETIC_PASSWORD")
	}

	if hostFlag != "" {
		cfg.Host = hostFlag
	}
	if userFlag != "" {
		cfg.User = userFlag
	}
	if passwordFlag != "" {
		cfg.Password = passwordFlag
	}

	return cfg, nil
}

// Validate checks if all required configuration fields are set.
func (c *Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host is required (set via flag, config file, or KEENETIC_HOST env var)")
	}
	if c.User == "" {
		return fmt.Errorf("user is required (set via flag, config file, or KEENETIC_USER env var)")
	}
	if c.Password == "" {
		return fmt.Errorf("password is required (set via flag, config file, or KEENETIC_PASSWORD env var)")
	}
	return nil
}

// SaveConfig saves configuration to the config file.
func SaveConfig(cfg *Config) error {
	configFile := getConfigFilePath()
	configDir := filepath.Dir(configFile)

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// GetConfigFilePath returns the path to the configuration file.
func GetConfigFilePath() string {
	return getConfigFilePath()
}

func getConfigFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".keenetic-routes-config.yaml"
	}
	return filepath.Join(homeDir, ".config", "keenetic-routes", "config.yaml")
}
