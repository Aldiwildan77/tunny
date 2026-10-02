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

There is no wildcard, suffix, CIDR, longest-prefix, policy-priority, or automatic provider selection. DNS changes after startup are not continuously tracked. A route can refer to a provider name that is absent from `providers`; the resulting dial fails when used.

## Configuration

`providers` maps a provider name to the address where the client reaches that provider. The address is used with the configured node transport.

`routes` maps a destination hostname to one provider name. The provider itself performs the final TCP dial.

See [providers.md](providers.md), [proxy.md](proxy.md), and [tunnel.md](tunnel.md).
