package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/creasty/defaults"
	"github.com/goccy/go-yaml"

	"github.com/go-playground/validator/v10"
)

type Config struct {
	Node          NodeConfig          `yaml:"node"`
	Tailscale     TailscaleConfig     `yaml:"tailscale"`
	Limiter       LimiterConfig       `yaml:"limiter"`
	Control       ControlConfig       `yaml:"control_plane"`
	Observability ObservabilityConfig `yaml:"observability"`

	// Requestor and Provider
	Proxy    ProxyConfig    `yaml:"proxy"`
	Provider ProviderConfig `yaml:"provider"`
	Tunnel   TunnelConfig   `yaml:"tunnel"`

	// Mapper
	Providers     map[string]string      `yaml:"providers" default:"{}"`
	Routes        map[string]string      `yaml:"routes" default:"{}"`
	RoutePolicies map[string]RoutePolicy `yaml:"route_policies"`
	Health        HealthConfig           `yaml:"health"`
}

type RoutePolicy struct {
	Mode      string   `yaml:"mode" validate:"omitempty,oneof=failover"`
	Providers []string `yaml:"providers"`
}

type HealthConfig struct {
	Enabled           bool          `yaml:"enabled" default:"true"`
	Interval          time.Duration `yaml:"interval" default:"5s"`
	Timeout           time.Duration `yaml:"timeout" default:"2s"`
	FailureThreshold  int           `yaml:"failure_threshold" default:"2"`
	RecoveryThreshold int           `yaml:"recovery_threshold" default:"3"`
}

type NodeConfig struct {
	Name      string `yaml:"name" validate:"required"`
	Transport string `yaml:"transport" default:"tailscale" validate:"oneof=direct tailscale"`
}

type TailscaleConfig struct {
	StateDir string `yaml:"state_dir" default:"~/.tunny"`
	AuthKey  string `yaml:"auth_key"`
}

type ProxyConfig struct {
	Listen     string `yaml:"listen" default:"127.0.0.1:1080"`
	HTTPListen string `yaml:"http_listen"`
}

type ProviderConfig struct {
	Listen string `yaml:"listen" default:":7070"`
}

type TunnelConfig struct {
	Enabled   bool   `yaml:"enabled" default:"true"`
	Interface string `yaml:"interface" default:"utun"`
	MTU       int    `yaml:"mtu" default:"1500"`
}

type LimiterConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
	Limit   int  `yaml:"limit" default:"10"`
}

type ControlConfig struct {
	Enabled    bool   `yaml:"enabled" default:"false"`
	Listen     string `yaml:"listen" default:"127.0.0.1:7071"`
	HTTPListen string `yaml:"http_listen" default:"127.0.0.1:7072"`
}

// An empty Endpoint lets the OTLP exporter fall back to OTEL_EXPORTER_OTLP_* env vars.
type ObservabilityConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Endpoint string `yaml:"endpoint"`
	Insecure bool   `yaml:"insecure"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	data = []byte(os.ExpandEnv(string(data)))

	var cfg Config

	// Defaults go first so YAML can override them. creasty/defaults only fills
	// zero values, so applying it afterwards would turn an explicit
	// `enabled: false` back into a `default:"true"`.
	if err := defaults.Set(&cfg); err != nil {
		return nil, fmt.Errorf("set default config: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
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
