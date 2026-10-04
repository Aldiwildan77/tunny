---
layout: default
title: Routing
---

# Routing

Tunny's current route table is a map from destination hostname to provider name:

```yaml
providers:
  japan: 100.64.0.10:7070

routes:
  example.com: japan
```

## Matching behavior

- Configured hostnames are lowercased.
- Tunny performs exact, case-insensitive hostname matching.
- At table creation, it resolves each hostname with `net.LookupHost` and records exact IP-to-provider mappings.
- A destination that matches neither a hostname nor a recorded IP uses the normal network path in proxy mode.
- Tunnel mode uses the resolved IPv4 addresses to install host routes on devices that implement the routing interface.
- Route changes through the control plane add or remove hostname mappings at runtime.

There is no wildcard, suffix, CIDR, or longest-prefix matching. DNS changes after startup are not continuously tracked. A route can refer to a provider name that is absent from `providers`; the resulting dial fails when used.

## Configuration

`providers` maps a provider name to the address where the client reaches that provider. The address is used with the configured node transport.

`routes` maps a destination hostname to one provider name. The provider itself performs the final TCP dial.

For ordered failover, use `route_policies`:

```yaml
route_policies:
  example.com:
    mode: failover
    providers:
      - japan
      - singapore
      - us
```

New connections try providers in this order. Providers marked unhealthy by the
active health checker are skipped. A failed connection attempt is recorded
immediately, while existing connections stay on their current provider.

## Provider health

Health checks are enabled by default and use Tunny's internal `PING` / `PONG`
protocol over the configured node transport. Consecutive failure and recovery
thresholds prevent a short network interruption from causing rapid flapping.

```yaml
health:
  enabled: true
  interval: 5s
  timeout: 2s
  failure_threshold: 2
  recovery_threshold: 3
```

See [providers.html](providers.html), [proxy.html](proxy.html), and [tunnel.html](tunnel.html).
