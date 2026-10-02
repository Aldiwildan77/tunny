# Providers

A provider is a Tunny process that acts as an internet egress node.

## Lifecycle

Run a provider with:

```bash
go run ./cmd/tunny provider -c example/proxy-provider/provider.yaml
```

The provider listens on `provider.listen` through the configured node. Direct nodes use `net.Listen`; Tailscale nodes use `tsnet.Server.Listen`. Shutdown closes the listener and active connections are allowed to finish or close.

## Protocol

The client connects to the configured provider address and sends:

```text
CONNECT example.com:443
```

The provider accepts only `CONNECT destination`. It returns `OK` after the destination TCP connection succeeds. It returns an `ERR ...` line for invalid requests, a missing destination, or a dial failure. After `OK`, both sides exchange raw TCP bytes.

This is Tunny's internal protocol. It is not SOCKS5 and it is not HTTP `CONNECT`.

## Configuration

```yaml
node:
  name: provider-1
  transport: direct

provider:
  listen: ":7070"
```

For a Tailscale provider, set `transport: tailscale` and configure `tailscale.state_dir` and, when needed, `tailscale.auth_key`.

## Limitations

The provider currently performs TCP destination dialing. Provider addresses are configured statically on the client. The provider does not expose health checks or a provider registry; external systems may monitor it, but automatic selection and failover are future work.
