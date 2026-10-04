---
layout: default
title: Providers
---

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

The client health checker uses the same provider listener with a separate
request:

```text
PING
```

A ready provider responds:

```text
PONG
```

Health state is tracked per provider. After the configured failure threshold,
the provider is excluded from new route-policy connections. It returns to the
candidate list after the configured number of consecutive successful checks.

## Configuration

```yaml
node:
  name: provider-1
  transport: direct

provider:
  listen: ":7070"
```

For a Tailscale provider, set `transport: tailscale` and configure `tailscale.state_dir` and, when needed, `tailscale.auth_key`.

## Health configuration

Health checks are configured on the client or serve node:

```yaml
health:
  enabled: true
  interval: 5s
  timeout: 2s
  failure_threshold: 2
  recovery_threshold: 3
```

Provider health is visible through `GET /v1/providers/health` when the control
plane is enabled.

## Limitations

The provider currently performs TCP destination dialing. Provider addresses are
configured statically on the client. Health checks cover protocol readiness and
reachability; they do not measure destination latency, bandwidth, or load.
