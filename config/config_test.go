package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "tunny.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestLoad(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name    string
		content string
		env     map[string]string
		want    *Config
		wantErr string
	}{
		{
			name: "full config",
			content: `
node:
  name: japan-node
  transport: direct

tailscale:
  state_dir: /var/lib/tunny
  auth_key: tskey-abc

proxy:
  listen: 0.0.0.0:1080

provider:
  listen: :8080

tunnel:
  interface: tun0
  mtu: 1400

providers:
  indonesia: indonesia:7070
  japan: japan:7070

routes:
  netflix.com: indonesia
  example.jp: japan
`,
			want: &Config{
				Node:      NodeConfig{Name: "japan-node", Transport: "direct"},
				Tailscale: TailscaleConfig{StateDir: "/var/lib/tunny", AuthKey: "tskey-abc"},
				Proxy:     ProxyConfig{Listen: "0.0.0.0:1080"},
				Provider:  ProviderConfig{Listen: ":8080"},
				Tunnel:    TunnelConfig{Interface: "tun0", MTU: 1400},
				Providers: map[string]string{"indonesia": "indonesia:7070", "japan": "japan:7070"},
				Routes:    map[string]string{"netflix.com": "indonesia", "example.jp": "japan"},
			},
		},
		{
			name: "minimal config applies defaults",
			content: `
node:
  name: minimal
`,
			want: &Config{
				Node:      NodeConfig{Name: "minimal", Transport: "tailscale"},
				Tailscale: TailscaleConfig{StateDir: filepath.Join(homeDir, ".tunny")},
				Proxy:     ProxyConfig{Listen: "127.0.0.1:1080"},
				Provider:  ProviderConfig{Listen: ":7070"},
				Tunnel:    TunnelConfig{Interface: "utun", MTU: 1500},
				Providers: map[string]string{},
				Routes:    map[string]string{},
			},
		},
		{
			name: "explicit tilde state_dir is expanded to home",
			content: `
node:
  name: tilde
tailscale:
  state_dir: ~/.tunny
`,
			want: &Config{
				Node:      NodeConfig{Name: "tilde", Transport: "tailscale"},
				Tailscale: TailscaleConfig{StateDir: filepath.Join(homeDir, ".tunny")},
				Proxy:     ProxyConfig{Listen: "127.0.0.1:1080"},
				Provider:  ProviderConfig{Listen: ":7070"},
				Tunnel:    TunnelConfig{Interface: "utun", MTU: 1500},
				Providers: map[string]string{},
				Routes:    map[string]string{},
			},
		},
		{
			name: "environment variables are expanded",
			content: `
node:
  name: ${TUNNY_TEST_NODE_NAME}
tailscale:
  auth_key: ${TUNNY_TEST_AUTHKEY}
`,
			env: map[string]string{
				"TUNNY_TEST_NODE_NAME": "env-node",
				"TUNNY_TEST_AUTHKEY":   "tskey-from-env",
			},
			want: &Config{
				Node:      NodeConfig{Name: "env-node", Transport: "tailscale"},
				Tailscale: TailscaleConfig{StateDir: filepath.Join(homeDir, ".tunny"), AuthKey: "tskey-from-env"},
				Proxy:     ProxyConfig{Listen: "127.0.0.1:1080"},
				Provider:  ProviderConfig{Listen: ":7070"},
				Tunnel:    TunnelConfig{Interface: "utun", MTU: 1500},
				Providers: map[string]string{},
				Routes:    map[string]string{},
			},
		},
		{
			name: "unset environment variable expands to empty and fails validation",
			content: `
node:
  name: ${TUNNY_TEST_UNSET_VAR}
`,
			wantErr: "validate config",
		},
		{
			name:    "invalid yaml",
			content: "node: [unclosed",
			wantErr: "parse config",
		},
		{
			name: "type mismatch",
			content: `
node:
  name: bad-mtu
tunnel:
  mtu: not-a-number
`,
			wantErr: "parse config",
		},
		{
			name: "missing node name",
			content: `
proxy:
  listen: 127.0.0.1:1080
`,
			wantErr: "validate config",
		},
		{
			name: "invalid transport",
			content: `
node:
  name: bad-transport
  transport: carrier-pigeon
`,
			wantErr: "validate config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, err := Load(writeConfig(t, tt.content))

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))

	require.ErrorContains(t, err, "read config")
	assert.ErrorIs(t, err, os.ErrNotExist)
	assert.Nil(t, got)
}

func TestLoad_ExampleConfig(t *testing.T) {
	t.Setenv("TS_AUTHKEY", "tskey-example")

	cfg, err := Load(filepath.Join("..", ".example.config.yaml"))
	require.NoError(t, err)

	assert.Equal(t, "japan-node", cfg.Node.Name)
	assert.Equal(t, "tskey-example", cfg.Tailscale.AuthKey)
	assert.Len(t, cfg.Providers, 3)
	assert.Len(t, cfg.Routes, 3)
}
