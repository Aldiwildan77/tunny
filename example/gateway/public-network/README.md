# Public network gateway

This example exposes HTTP and SOCKS5 ingress on the host's network
interfaces. It is suitable for testing from another device on the same
private Wi-Fi or wired network. `serve` also starts transparent TUN ingress by
default, so run it with administrator privileges.

This is not an internet-public proxy. Do not port-forward ports `8080` or
`1080`, and do not use this configuration on an untrusted network.

## Start the provider

From the repository root:

```bash
go run ./cmd/tunny provider -c example/gateway/public-network/provider.yaml
```

## Start serve mode

In another terminal:

```bash
sudo go run ./cmd/tunny serve -c example/gateway/public-network/gateway.yaml
```

Find the host's private address on macOS:

```bash
ipconfig getifaddr en0
```

Try `en1` if Wi-Fi uses another interface. Configure another device with the
returned address, for example `192.168.1.25`:

- HTTP proxy: `192.168.1.25`, port `8080`
- SOCKS5 proxy: `192.168.1.25`, port `1080`

The devices must be on the same network. Allow the Go process through the
macOS firewall on the private network. Client isolation on the Wi-Fi access
point can prevent devices from reaching the host.

To run proxy ingress without TUN, set this in the example config:

```yaml
tunnel:
	enabled: false
```