# Tunny

**Programmable egress routing for multi-node networks.**

Tunny routes selected outbound traffic through configured provider nodes. A provider is a reachable node that can connect to the internet on the client's behalf. Tunny can reach providers over direct networking or Tailscale.

Tailscale answers **how can one node reach another?** Tunny answers **which provider should handle this outbound flow?**

```text
Client
  |
  v
Tunny
  |
  +-- route matching --> provider address
                              |
                         transport
                              |
                         Provider
                              |
                          Internet
```

## Why Tunny?

Tunny sits beside other networking tools rather than replacing them:

| Concern | Typical tools | Tunny's role |
| --- | --- | --- |
| Node connectivity | Tailscale, WireGuard | Uses direct networking or Tailscale to reach a provider. |
| Endpoint selection | GSLB, GeoDNS | Routes a flow according to configured destination-to-provider rules. |
| Application proxying | HAProxy, Nginx, Envoy | Provides generic HTTP and SOCKS5 entry points and a small provider forwarding protocol. |
| Network path selection | BGP, ECMP | Can be part of the network that makes providers reachable. |
| Internet egress selection | Egress gateways and VPN exits | Treats reachable machines as provider nodes and sends selected traffic through one. |

Tunny is not primarily a VPN, generic load balancer, GSLB, or IP-changing service. HTTP, SOCKS5, and TUN are ingress modes for applying its egress-routing behavior.

## Use cases

- **Multiple VPS egress points:** map selected destinations to VPS instances in Japan, Singapore, the US, or another reachable location.
- **Geo-specific egress:** deliberately send a destination through a chosen provider instead of selecting the nearest endpoint.
- **VPN-like system routing:** use transparent TUN ingress for selected system traffic where the platform's TUN implementation supports routing.
- **Application routing:** use HTTP or SOCKS5 ingress when applications or devices support proxy settings.
- **Multi-provider experiments:** operate several provider nodes and combine Tunny with external health checks, DNS, BGP, or ECMP. Automatic provider pools and health-aware selection are not implemented by core Tunny today.

## How it works

1. A client sends traffic to `serve`, which starts HTTP/SOCKS5 ingress and transparent TUN ingress together by default.
2. Tunny identifies the destination hostname or IP address.
3. The route table checks for an exact configured hostname or resolved IP match.
4. Unmatched traffic uses the normal network path.
5. Matched traffic is sent to the configured provider address.
6. The provider connects to the destination over TCP and forwards bytes in both directions.

The current route table maps one hostname to one provider name. It does not implement wildcards, CIDR matching, longest-prefix matching, automatic provider selection, or failover.

## Components

- **Serve:** combines HTTP, SOCKS5, and transparent TUN ingress using one route table and provider map. Transparent mode requires elevated privileges and can be disabled with `tunnel.enabled: false`.
- **Proxy:** compatibility mode exposing only the SOCKS5 listener.
- **Tunnel:** compatibility mode exposing only the transparent TUN path.
- **Provider:** accepts Tunny's internal `CONNECT host:port` request and dials the destination.
- **Node:** supplies direct or Tailscale `Dial` and `Listen` operations.
- **Control plane:** exposes gRPC and an HTTP/JSON gateway for status and route changes.
- **Data plane:** carries the actual proxy, provider, and tunnel traffic.

See [docs/architecture.md](docs/architecture.md) and [docs/concepts.md](docs/concepts.md).

## Quick start

The smallest working example uses a local provider and a local SOCKS5 proxy:

```bash
go run ./cmd/tunny provider -c example/proxy-provider/provider.yaml
```

In another terminal:

```bash
go run ./cmd/tunny proxy -c example/proxy-provider/client.yaml
```

Then test a routed request:

```bash
curl --proxy socks5h://127.0.0.1:1080 --connect-timeout 5 --max-time 10 https://example.com
```

The example routes `example.com` to `127.0.0.1:7070`; unmatched destinations use the client's normal network connection. See [docs/examples/single-provider.md](docs/examples/single-provider.md).

Build the binary with:

```bash
make build
./tunny --help
```

The global flags are `--config`, `--daemon`, `--pid`, and `--log`. The commands are `serve`, `proxy`, `tunnel`, `provider`, `daemon status`, and `daemon stop`.

## Docker and Kubernetes

Tunny is a normal Go process, so it can run as a container in a distributed
network. A typical deployment separates the roles:

```text
Client or application pod
     |
     v
   Tunny proxy/tunnel
     |
     v
   provider service or node
     |
     v
  Internet
```

### Docker Compose

The repository includes a multi-node Docker Compose lab at
[example/bgp-multi-node/docker-compose.yaml](example/bgp-multi-node/docker-compose.yaml).
It builds three Tunny provider containers alongside FRR and gdnsd. The
example is an experiment that combines several networking layers; it is not a
general-purpose Tunny cluster manager.

Start that lab from the example directory with:

```bash
cd example/bgp-multi-node
docker compose up --build
```

The Compose file exposes the provider listeners on host ports `17071`,
`17072`, and `17073`. The client configuration is
[client/tunny.yaml](example/bgp-multi-node/client/tunny.yaml), and the Tunny
client must be run separately as described by the example.

### Kubernetes

Tunny can be packaged into Kubernetes Deployments, StatefulSets, or Jobs by
building a container image that runs one of the existing commands:

```text
tunny proxy   -c /etc/tunny/config.yaml
tunny provider -c /etc/tunny/config.yaml
tunny tunnel  -c /etc/tunny/config.yaml
```

Use a Kubernetes `Service` or another reachable network address for provider
listeners, then reference that address in the client's `providers` map. Store
the YAML configuration in a ConfigMap and credentials such as a Tailscale auth
key in a Secret.

Proxy deployments normally need only a TCP service for the SOCKS5 listener.
Provider deployments need a TCP service for the provider listener. Tunnel
deployments require platform-specific TUN access and network privileges; the
current tunnel implementation is not a drop-in Kubernetes CNI or cluster
overlay. See [docs/tunnel.md](docs/tunnel.md) for its current platform
limitations.

This repository does not currently include a maintained Docker image,
Kubernetes manifests, Helm chart, operator, service discovery, or automatic
provider pool. Those are deployment work to build around the Tunny binary, not
implemented features of the core routing logic.

## Control plane

Enable it in a config file:

```yaml
control_plane:
  enabled: true
  listen: 127.0.0.1:7071
  http_listen: 127.0.0.1:7072
```

The HTTP gateway exposes `GET /v1/status`, `GET /v1/routes`, `POST /v1/routes`, and `DELETE /v1/routes/{hostname}`. See [docs/control-plane.md](docs/control-plane.md).

## Documentation

The same documentation is synchronized to the [GitHub Wiki](https://github.com/Aldiwildan77/tunny/wiki) when changes land on `master`.

- [Concepts](docs/concepts.md)
- [Architecture](docs/architecture.md)
- [Routing](docs/routing.md)
- [Providers](docs/providers.md)
- [Transports](docs/transports.md)
- [Tunnel mode](docs/tunnel.md)
- [Proxy mode](docs/proxy.md)
- [Control plane](docs/control-plane.md)
- [Multi-node operation](docs/multi-node.md)
- [Single-provider example](docs/examples/single-provider.md)
- [Tailscale example](docs/examples/tailscale.md)
- [Multi-provider example](docs/examples/multi-provider.md)
- [Daemon operation](DAEMON.md)
- [Development](docs/development.md)

## Status

**Implemented:** direct and Tailscale nodes, exact hostname/IP route matching, SOCKS5 proxying, the provider forwarding protocol, control-plane status and route APIs, and platform-specific daemon process handling.

**Experimental:** the BGP/gdnsd topology under [example/bgp-multi-node](example/bgp-multi-node) combines Tunny with FRR and gdnsd. It is an example environment, not a built-in Tunny routing algorithm.

**Planned or not implemented:** automatic provider pools, health-aware selection, latency/load-aware routing, wildcard/CIDR route matching, IPv6 tunnel routing, and automatic failover in the core route table.

## Related technologies

Tailscale, WireGuard, WARP, HAProxy, Nginx, Envoy, GSLB/GeoDNS, BGP, ECMP, anyhop, and global-egress address adjacent layers or related egress workflows. Tunny's intended abstraction is a route-aware client that chooses a reachable provider node for selected outbound traffic; it does not claim to replace those systems.

## Repository structure

Tunny uses a flat repository structure to keep core platform development
simple. The main packages stay easy to find at the top level, while related
capabilities remain separated into focused packages.
