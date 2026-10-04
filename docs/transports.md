---
layout: default
title: Transports
---

# Transports

The node transport is separate from route and provider selection. It answers how the client reaches the configured provider address.

## Direct

```yaml
node:
  name: local-client
  transport: direct
```

The direct node uses the operating system network through `net.Dialer` and `net.Listen`. Use it when provider addresses are directly reachable.

## Tailscale

```yaml
node:
  name: client
  transport: tailscale

tailscale:
  state_dir: ~/.tunny
  auth_key: ${TS_AUTHKEY}
```

The Tailscale node uses `tailscale.com/tsnet`. Its hostname, state directory, and optional auth key are taken from configuration. Tailscale supplies node connectivity; Tunny still chooses which configured provider address handles a matched destination.

## Other transports

The `node.Node` interface leaves room for additional transports, but no others are implemented in this repository. WireGuard, BGP, and ECMP may be used by surrounding network infrastructure without being Tunny transports here.
