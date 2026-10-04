---
layout: default
title: Concepts
---

# Concepts

## Client

The client is the Tunny process accepting traffic. `serve` is the combined
mode and starts HTTP/SOCKS5 proxy ingress plus transparent TUN ingress by
default. `proxy` and `tunnel` remain compatibility modes for running one
ingress type. The client owns the route table and provider address map.

## Provider

A provider is a Tunny process that listens for client connections and connects to the final internet destination. The provider is the egress point.

## Egress

Egress is the provider-to-destination connection. The configured node transport applies to the client-to-provider leg; the provider uses TCP to reach the destination.

## Route

A route maps a hostname to a provider name, for example `example.com: local`. At startup, Tunny resolves configured hostnames and also records exact resolved IP mappings.

## Provider pool

A provider pool is an ordered list of providers for one route. Configure one with
`route_policies` and `mode: failover`; health-aware selection skips unhealthy
providers for new connections and preserves the configured order.

## Transport

A transport is the node-to-node connectivity mechanism used by Tunny's `Node` interface. Current choices are `direct` and `tailscale`.

## Serve

Serve mode combines the HTTP, SOCKS5, and transparent TUN ingress paths. It
uses one route table and provider map for all enabled flows. Transparent mode
requires platform support and elevated privileges; set `tunnel.enabled` to
`false` for proxy-only serve operation. See [serve.html](serve.html).

## Tunnel

Tunnel mode uses a TUN device and a gVisor IPv4 stack to process system
traffic. It is also the transparent ingress used by serve mode. See
[tunnel.html](tunnel.html).

## Proxy

Proxy mode exposes a SOCKS5 listener for application traffic. See [proxy.html](proxy.html).

## Control plane

The control plane is an optional gRPC server with an HTTP/JSON gateway. It reports status and changes routes at runtime.

## Data plane

The data plane is the traffic path: client input, route matching, provider connection, provider egress, and response forwarding.
