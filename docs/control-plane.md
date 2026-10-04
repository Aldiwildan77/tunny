---
layout: default
title: Control plane
---

# Control plane

The control plane is optional and disabled by default. Enable it in YAML:

```yaml
control_plane:
  enabled: true
  listen: 127.0.0.1:7071
  http_listen: 127.0.0.1:7072
```

Every data-plane command can start the control plane alongside its normal service. The implementation is defined in [control-plane/control.proto](../control-plane/control.proto) and [control-plane/server.go](../control-plane/server.go).

## APIs

| Method | HTTP endpoint | Purpose |
| --- | --- | --- |
| `GetStatus` | `GET /v1/status` | Returns mode, node name, transport, uptime, and active connection count. |
| `ListRoutes` | `GET /v1/routes` | Lists the current hostname-to-provider mappings. |
| `SetRoute` | `POST /v1/routes` | Adds or updates a hostname-to-provider mapping. |
| `DeleteRoute` | `DELETE /v1/routes/{hostname}` | Removes a hostname mapping. |
| `ListProviderHealth` | `GET /v1/providers/health` | Reports provider state, counters, timestamps, and the latest error. |

Example:

```bash
curl http://127.0.0.1:7072/v1/status
curl http://127.0.0.1:7072/v1/routes
curl http://127.0.0.1:7072/v1/providers/health
curl -X POST http://127.0.0.1:7072/v1/routes \
  -H 'Content-Type: application/json' \
  -d '{"hostname":"example.com","provider":"local"}'
curl -X DELETE http://127.0.0.1:7072/v1/routes/example.com
```

The gRPC service exposes the same five operations. Use [control.proto](../control-plane/control.proto) with `grpcurl` when direct gRPC access is preferred.

## Current behavior

`SetRoute` trims and lowercases the hostname and requires non-empty hostname and provider values. It does not verify that the provider exists in the static `providers` map. `DeleteRoute` accepts any non-empty hostname and returns success even when no mapping existed. Provider addresses and listener settings cannot be changed through this API.

The reported active connection count is currently zero because the command paths do not update the control-plane counter.
