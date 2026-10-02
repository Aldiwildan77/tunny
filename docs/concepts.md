# Concepts

## Client

The client is the Tunny process accepting traffic in `proxy` or `tunnel` mode. It owns the route table and provider address map.

## Provider

A provider is a Tunny process that listens for client connections and connects to the final internet destination. The provider is the egress point.

## Egress

Egress is the provider-to-destination connection. The configured node transport applies to the client-to-provider leg; the provider uses TCP to reach the destination.

## Route

A route maps a hostname to a provider name, for example `example.com: local`. At startup, Tunny resolves configured hostnames and also records exact resolved IP mappings.

## Provider pool

A provider pool would mean several interchangeable providers for one route. The configuration currently maps each route to one provider name. Pool selection and automatic failover are planned, not implemented.

## Transport

A transport is the node-to-node connectivity mechanism used by Tunny's `Node` interface. Current choices are `direct` and `tailscale`.

## Tunnel

Tunnel mode uses a TUN device and a gVisor IPv4 stack to process system traffic. See [tunnel.md](tunnel.md).

## Proxy

Proxy mode exposes a SOCKS5 listener for application traffic. See [proxy.md](proxy.md).

## Control plane

The control plane is an optional gRPC server with an HTTP/JSON gateway. It reports status and changes routes at runtime.

## Data plane

The data plane is the traffic path: client input, route matching, provider connection, provider egress, and response forwarding.
