package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/creasty/defaults"
	"gopkg.in/yaml.v3"

	"github.com/go-playground/validator/v10"
)

type Config struct {
	Node      NodeConfig      `yaml:"node"`
	Tailscale TailscaleConfig `yaml:"tailscale"`

	// Requestor and Provider
	Proxy    ProxyConfig    `yaml:"proxy"`
	Provider ProviderConfig `yaml:"provider"`
	Tunnel   TunnelConfig   `yaml:"tunnel"`

	// Mapper
	Providers map[string]string `yaml:"providers" default:"{}"`
	Routes    map[string]string `yaml:"routes" default:"{}"`
}

type NodeConfig struct {
	Name      string `yaml:"name" validate:"required"`
	Transport string `yaml:"transport" default:"tailscale" validate:"oneof=direct tailscale"`
}

type TailscaleConfig struct {
	StateDir string `yaml:"state_dir" default:"~/.tunny"`
	AuthKey  string `yaml:"auth_key" validate:"required"`
}

type ProxyConfig struct {
	Listen string `yaml:"listen" default:"127.0.0.1:1080"`
}

type ProviderConfig struct {
	Listen string `yaml:"listen" default:":7070"`
}

type TunnelConfig struct {
	Interface string `yaml:"interface" default:"utun"`
	MTU       int    `yaml:"mtu" default:"1500"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	data = []byte(os.ExpandEnv(string(data)))

	var cfg Config

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := defaults.Set(&cfg); err != nil {
		return nil, fmt.Errorf("set default config: %w", err)
	}

	if err := validator.New().Struct(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	if cfg.Tailscale.StateDir == "~/.tunny" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get user home dir: %w", err)
		}
		cfg.Tailscale.StateDir = filepath.Join(homeDir, ".tunny")
	}

	return &cfg, nil
}
