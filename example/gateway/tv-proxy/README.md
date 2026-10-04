# TV HTTP proxy

This example exposes a standard HTTP proxy for a TV or other device that
supports manual HTTP proxy settings. It does not contain TV-specific routing,
streaming-service logic, TLS interception, or application inspection.
Serve mode also starts transparent TUN ingress by default; disable it in the
config if this host should provide only the TV's HTTP proxy.

## Start the provider

From the repository root:

```bash
go run ./cmd/tunny provider -c example/gateway/tv-proxy/provider.yaml
```

## Start serve mode

In another terminal:

```bash
sudo go run ./cmd/tunny serve -c example/gateway/tv-proxy/gateway.yaml
```

Find the computer's private Wi-Fi address:

```bash
ipconfig getifaddr en0
```

Use `en1` if needed. In the TV network settings, configure:

- Proxy type: HTTP
- Proxy host: the computer's private address, such as `192.168.1.25`
- Proxy port: `8080`

The TV and computer must be on the same Wi-Fi network. Allow the Go process
through the macOS firewall on the private network. This listener binds to all
interfaces for LAN testing; do not expose it through a router or public IP.

Proxy-only configuration:

```yaml
tunnel:
	enabled: false
```