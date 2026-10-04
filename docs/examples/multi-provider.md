---
layout: default
title: Multi-provider example
---

# Multi-provider configuration

The core configuration can name several providers and map different destinations to them:

```yaml
node:
  name: client
  transport: direct

providers:
  japan: 192.0.2.10:7070
  singapore: 192.0.2.20:7070
  us: 192.0.2.30:7070

routes:
  service.jp: japan
  service.sg: singapore
  service.us: us
```

The addresses above are documentation placeholders. Replace them with reachable provider addresses before running the client.

The repository's concrete multi-node configuration is [example/bgp-multi-node/client/tunny.yaml](../../example/bgp-multi-node/client/tunny.yaml), which maps two hostnames to `id1`; the surrounding example supplies FRR and gdnsd containers but requires a separately run Tunny client.

Multiple configured providers do not form an automatic pool. Health checks, failover, latency selection, and load balancing remain external or planned behavior.
