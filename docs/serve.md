---
layout: default
title: Serve mode
---

# Serve mode

Serve mode runs the configured client ingress services together: HTTP forward
proxying, including CONNECT, SOCKS5, and transparent TUN ingress. The internal
gateway package provides the shared ingress lifecycle and routing composition.

```bash
sudo go run ./cmd/tunny serve -c config.yaml
```

The default serve listeners are `127.0.0.1:8080` for HTTP and
`127.0.0.1:1080` for SOCKS5. Keep these listeners on loopback unless an
external access-control layer protects them; exposing an unauthenticated
proxy publicly can create an open proxy. Serve mode does not terminate TLS,
inspect application payloads, or implement device-specific behavior.

Serve starts proxy and transparent tunnel ingress together by default. It
requires TUN/device privileges, so run it with `sudo` where the platform
requires elevated network access.

Configure listeners independently:

```yaml
proxy:
  listen: 127.0.0.1:1080
  http_listen: 127.0.0.1:8080

tunnel:
  enabled: true
  interface: utun
  mtu: 1500
```

All enabled ingress modes use the same route table and provider map. Set
`tunnel.enabled` to `false` for proxy-only serve mode. Transparent mode
remains platform- and privilege-dependent, and provider-side UDP forwarding is
still limited by the existing provider path.

## Health-aware failover

Serve mode starts the provider health checker alongside its ingress services.
Use `route_policies` when a destination should have an ordered provider list:

```yaml
providers:
  japan: 100.64.0.10:7070
  singapore: 100.64.0.11:7070

route_policies:
  example.com:
    mode: failover
    providers: [japan, singapore]

health:
  enabled: true
  interval: 5s
  timeout: 2s
  failure_threshold: 2
  recovery_threshold: 3
```

New connections prefer `japan`, skip it after health failures, and try
`singapore` automatically. Existing TCP connections are not moved between
providers. Recovery checks return a provider to the ordered candidate list.

## Routing behavior

HTTP and SOCKS5 ingress use the shared route table. Exact route matches are
sent to the configured provider. Unmatched destinations use the normal local
network path. For matched traffic, the client connects to the provider through
the configured node transport and the provider connects to the destination.

Test the HTTP listener with:

```bash
curl --proxy http://127.0.0.1:8080 http://example.com
```

Test the SOCKS5 listener with:

```bash
curl --proxy socks5h://127.0.0.1:1080 https://example.com
```

The gateway examples for private networks and TVs are under
[example/gateway](https://github.com/Aldiwildan77/tunny/tree/master/example/gateway).
