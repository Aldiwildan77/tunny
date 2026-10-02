# Development

## Build and test

```bash
make build
make test
make fmt
make check
```

`make build` creates `./tunny` from `./cmd/tunny`. `make test` runs `go test ./...`. Tunnel tests create real TUN devices and may require platform support and elevated privileges.

## Repository layout

- `cmd/tunny`: Cobra CLI and process startup.
- `config`: YAML loading, environment expansion, defaults, and validation.
- `node`: direct and Tailscale connectivity implementations.
- `route`: exact hostname/IP route table.
- `proxy`: SOCKS5 server and route-aware dialer.
- `provider`: provider listener and internal forwarding protocol.
- `tunnel`: TUN device and gVisor packet forwarding.
- `control-plane`: gRPC service and HTTP/JSON gateway.
- `example`: local, Tailscale-oriented, and multi-node configurations.

## Configuration notes

Configuration is YAML. Environment variables are expanded before parsing, so values such as `${TS_AUTHKEY}` can be used. Defaults and currently validated fields are documented in [concepts.md](concepts.md) and the example files.

## Release

Tags matching `v*` trigger the GitHub Actions release workflow. GoReleaser builds Linux, macOS, and Windows binaries for amd64 and arm64, then creates archives and package artifacts configured in `.goreleaser.yaml`.

## Adding behavior

Keep transport connectivity behind `node.Node`, route behavior in `route.Table`, and data-plane forwarding in the relevant package. When documenting a new capability, distinguish verified behavior from experimental integrations and planned selection algorithms.
