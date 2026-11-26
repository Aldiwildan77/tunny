---
layout: default
title: Observability
---

# Observability

Tunny exports metrics for every command and feature over OTLP/gRPC: connections, routing, the HTTP proxy, the TUN device, providers, health checks, the control plane, and the Go runtime. It also writes structured logs for every routing decision. Observability is disabled by default. Enable it once at the top level of the config; it applies to whichever command runs (`proxy`, `provider`, `tunnel`, or `serve`):

```yaml
observability:
  enabled: true
  endpoint: localhost:4317
  insecure: true
```

| Field | Default | Meaning |
| --- | --- | --- |
| `enabled` | `false` | Start the OpenTelemetry metric and trace exporters. |
| `endpoint` | empty | OTLP/gRPC collector address. When empty, the exporter reads `OTEL_EXPORTER_OTLP_ENDPOINT` and falls back to `localhost:4317`. |
| `insecure` | `false` | Use plaintext gRPC. Leave it `false` for a collector that serves TLS. |

Standard `OTEL_*` environment variables still apply. Use them to give one process different settings without changing the shared config file, for example `OTEL_EXPORTER_OTLP_ENDPOINT` in a single systemd unit. `OTEL_RESOURCE_ATTRIBUTES` and `OTEL_SERVICE_NAME` override the resource attributes below.

## Resource attributes

| Attribute | Value |
| --- | --- |
| `service.namespace` | `tunny` |
| `service.name` | `tunny-<command>`, for example `tunny-proxy` or `tunny-serve` |
| `service.instance.id` | `node.name` |
| `service.version` | Build version. Release binaries and `make build` set it; `go run` reports `dev`. |

## Metrics by command

Every command exports the shared metrics. Each command also exports the metrics for the features it runs:

| Command | Ingress (`tunny.ingress`) | Feature metrics |
| --- | --- | --- |
| `proxy` | `socks5` | Route, direct dial |
| `tunnel` | `tun` | Route, direct dial, TUN device |
| `serve` | `socks5`, `http`, `tun` (whichever are enabled) | Route, direct dial, HTTP proxy, TUN device |
| `provider` | `provider` | Provider |

Shared by all commands: connections, Go runtime, route table size, provider health, and control plane (when `control_plane.enabled`). Filter by `service.name` (Prometheus `job`) to separate commands, and by `service.instance.id` (`instance`) to separate nodes.

### Connections (all commands)

Every accepted client connection is tracked: SOCKS5 and HTTP proxy clients, TUN TCP and UDP flows, and tunnels arriving at a provider.

| Metric | Type | Attributes |
| --- | --- | --- |
| `tunny.connection.active` | UpDownCounter, `{connection}` | `tunny.ingress`, `network.transport` |
| `tunny.connection.duration` | Histogram, seconds | `tunny.ingress`, `network.transport` |
| `tunny.connection.io` | Counter, bytes | `tunny.ingress`, `network.transport`, `network.io.direction` |

`network.io.direction` is from Tunny's side of the client connection: `receive` is bytes read from the client, `transmit` is bytes written to it. Bytes are counted as they flow, so long-lived connections show up before they close. The control plane `GetStatus` `active_connections` field reports the same open-connection count.

### Route (`proxy`, `tunnel`, `serve`)

Recorded where a destination is matched to a route and dialled through a provider. SOCKS5, HTTP proxy, and TUN ingress all share this path. Durations measure connection setup only: reaching the provider, sending `CONNECT`, and receiving the reply.

| Metric | Type | Attributes |
| --- | --- | --- |
| `tunny.route.request.duration` | Histogram, seconds | `tunny.route`, `network.transport`, `tunny.outcome`, `tunny.provider` (success), `error.type` (error) |
| `tunny.route.provider.attempt.duration` | Histogram, seconds | `tunny.route`, `network.transport`, `tunny.provider`, `tunny.outcome`, `error.type` (error) |
| `tunny.direct.dial.duration` | Histogram, seconds | `network.transport`, `tunny.outcome`, `error.type` (error) |

`tunny.route.request.duration` is recorded once per routed connection across all failover attempts. `tunny.route.provider.attempt.duration` is recorded once per provider tried. `tunny.direct.dial.duration` covers destinations that match no route.

`tunny.route` is the configured route hostname. Destinations matched by IP, as in tunnel mode, are reported under the hostname the IP was resolved from.

| Route `error.type` | Cause |
| --- | --- |
| `provider_not_found` | The route names a provider missing from the `providers` map. |
| `provider_unreachable` | The node could not dial the provider. |
| `handshake` | Writing `CONNECT` or reading the reply failed. |
| `provider_rejected` | The provider answered `ERR`, usually because it could not reach the destination. |
| `no_healthy_provider` | Health checks removed every candidate provider. |
| `timeout` | The dial or handshake timed out. |
| `canceled` | The client went away before the connection was established. |

Request-level errors carry the `error.type` of the last provider attempt.

### HTTP proxy (`serve` with `proxy.http_listen`)

| Metric | Type | Attributes |
| --- | --- | --- |
| `tunny.http.request.duration` | Histogram, seconds | `http.request.method`, `http.response.status_code` |

`502` means Tunny could not reach the destination; other status codes come from the origin. A `CONNECT` request lasts for the whole tunnel, so its duration is the tunnel lifetime. Unknown methods are reported as `_OTHER`.

### TUN device (`tunnel`, `serve` with `tunnel.enabled`)

| Metric | Type | Attributes |
| --- | --- | --- |
| `tunny.tunnel.packets` | Counter, `{packet}` | `network.io.direction`, `tunny.outcome`, `error.type` (`device_write`) |
| `tunny.tunnel.io` | Counter, bytes | `network.io.direction` |
| `tunny.tunnel.forward.errors` | Counter, `{flow}` | `network.transport`, `error.type` (`create_endpoint`, `dial`) |

`receive` is packets read from the TUN device (traffic leaving the host); `transmit` is packets written back to it. A failed device write drops the packet.

### Provider (`provider`)

| Metric | Type | Attributes |
| --- | --- | --- |
| `tunny.provider.requests` | Counter, `{request}` | `tunny.request.type`: `connect`, `ping`, `invalid` |
| `tunny.provider.target.dial.duration` | Histogram, seconds | `tunny.outcome`, `error.type` (error) |

The target dial is the far end of a `provider_rejected` route error. Its `error.type` is one of `dns`, `refused`, `unreachable`, `timeout`, `canceled`, or `dial`.

### Provider health (all commands)

| Metric | Type | Attributes |
| --- | --- | --- |
| `tunny.health.check.duration` | Histogram, seconds | `tunny.provider`, `tunny.outcome`, `error.type` (error) |
| `tunny.health.transitions` | Counter, `{transition}` | `tunny.provider`, `tunny.health.state` (new state) |
| `tunny.health.healthy` | Gauge | `tunny.provider`; `1` healthy, `0` unhealthy |

Check `error.type` is one of `provider_unreachable`, `handshake`, `unexpected_response`, `timeout`, or `canceled`. State also changes from failures seen by live traffic, so transitions can occur with `health.enabled: false`.

### Route table (all commands)

| Metric | Type | Attributes |
| --- | --- | --- |
| `tunny.route.count` | Gauge, `{entry}` | `tunny.route.kind`: `hostname`, `policy`, `ip` |

### Control plane (when enabled)

| Metric | Type | Attributes |
| --- | --- | --- |
| `tunny.control.rpc.duration` | Histogram, seconds | `rpc.method`, `rpc.grpc.status_code` |
| `tunny.control.http.duration` | Histogram, seconds | `http.request.method`, `http.route`, `http.response.status_code` |

`http.route` collapses `/v1/routes/<hostname>` to `/v1/routes/{hostname}`.

### Go runtime (all commands)

From `go.opentelemetry.io/contrib/instrumentation/runtime`: `go.goroutine.count`, `go.memory.used`, `go.memory.allocated`, `go.memory.allocations`, `go.memory.gc.goal`, `go.processor.limit`, and `go.config.gogc`.

## Structured logs

Route decisions are logged with Go's `log/slog` as `key=value` pairs, in the same output as the rest of the process log:

```text
2026/11/26 10:15:02 INFO route matched address=netflix.com:443 route=netflix.com candidates="[indonesia singapore]" healthy="[indonesia singapore]"
2026/11/26 10:15:02 WARN route provider attempt failed address=netflix.com:443 route=netflix.com provider=indonesia attempt=1 error_type=provider_unreachable error="dial tcp 100.64.0.2:7070: connect: connection refused" duration=3.1ms
2026/11/26 10:15:02 INFO route connected address=netflix.com:443 route=netflix.com provider=singapore attempt=2 duration=41.7ms
2026/11/26 10:15:09 ERROR route failed address=example.jp:443 route=example.jp attempts=1 error_type=provider_rejected error="provider error: ERR dial tcp: i/o timeout" duration=5.0s
```

| Message | Level | Meaning |
| --- | --- | --- |
| `route matched` | INFO | A destination matched a route. Lists configured and healthy candidates. |
| `route provider attempt failed` | WARN | One provider failed; Tunny tries the next candidate. |
| `route connected` | INFO | Connection established, with the provider that served it. |
| `route failed` | ERROR | Every candidate failed. |

Top failing routes from a log file:

```bash
grep 'route failed' /var/log/tunny.log | grep -o 'route=[^ ]*' | sort | uniq -c | sort -rn | head
```

## Collector setup

Point Tunny at an OpenTelemetry Collector and expose the metrics to Prometheus:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

exporters:
  prometheus:
    endpoint: 0.0.0.0:8889

service:
  pipelines:
    metrics:
      receivers: [otlp]
      exporters: [prometheus]
```

OTLP names are translated for Prometheus: dots become underscores, and unit suffixes are added (`_seconds`, `_bytes`, `_total` for counters). For example, `tunny.route.request.duration` becomes `tunny_route_request_duration_seconds` and `tunny.connection.io` becomes `tunny_connection_io_bytes_total`. Attributes become labels the same way (`tunny.route` becomes `tunny_route`). `job` is `tunny/tunny-<command>` and `instance` is the node name. These names assume the collector's default translation.

## Queries

Top 10 routes by error rate:

```promql
topk(10,
  sum by (tunny_route) (rate(tunny_route_request_duration_seconds_count{tunny_outcome="error"}[5m]))
  /
  sum by (tunny_route) (rate(tunny_route_request_duration_seconds_count[5m]))
)
```

Success rate per route:

```promql
sum by (tunny_route) (rate(tunny_route_request_duration_seconds_count{tunny_outcome="success"}[5m]))
/
sum by (tunny_route) (rate(tunny_route_request_duration_seconds_count[5m]))
```

Top 10 slowest routes by p95 connection setup time:

```promql
topk(10,
  histogram_quantile(0.95,
    sum by (tunny_route, le) (rate(tunny_route_request_duration_seconds_bucket{tunny_outcome="success"}[5m]))
  )
)
```

Failures by route and cause:

```promql
sum by (tunny_route, error_type) (rate(tunny_route_request_duration_seconds_count{tunny_outcome="error"}[5m]))
```

Provider error rate per route, which shows the provider behind a failover:

```promql
sum by (tunny_route, tunny_provider) (rate(tunny_route_provider_attempt_duration_seconds_count{tunny_outcome="error"}[5m]))
/
sum by (tunny_route, tunny_provider) (rate(tunny_route_provider_attempt_duration_seconds_count[5m]))
```

Failovers per second per route, counted as provider attempts beyond the first:

```promql
sum by (tunny_route) (rate(tunny_route_provider_attempt_duration_seconds_count[5m]))
-
sum by (tunny_route) (rate(tunny_route_request_duration_seconds_count[5m]))
```

Open connections per command and ingress:

```promql
sum by (job, tunny_ingress, network_transport) (tunny_connection_active)
```

Client throughput in bytes per second:

```promql
sum by (job, tunny_ingress, network_io_direction) (rate(tunny_connection_io_bytes_total[5m]))
```

Destinations a provider cannot reach, by cause:

```promql
sum by (instance, error_type) (rate(tunny_provider_target_dial_duration_seconds_count{tunny_outcome="error"}[5m]))
```

Flapping providers (state changes in the last hour):

```promql
sum by (tunny_provider) (increase(tunny_health_transitions_total[1h]))
```

Providers currently unhealthy:

```promql
tunny_health_healthy == 0
```

HTTP proxy responses by status code:

```promql
sum by (http_response_status_code) (rate(tunny_http_request_duration_seconds_count[5m]))
```

TUN packets dropped on write:

```promql
sum by (instance) (rate(tunny_tunnel_packets_total{tunny_outcome="error"}[5m]))
```

Goroutines per process, useful for spotting leaked flows:

```promql
go_goroutine_count
```

A baseline dashboard needs four panels: error rate and p95 setup time per route (the first and third queries as time series), failures by cause (stacked), and provider error rate per route (table).
