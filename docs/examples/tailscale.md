---
layout: default
title: Tailscale example
---

# Tailscale transport

Tailscale is an optional transport for reaching provider nodes. It does not choose the egress provider; the Tunny route table still maps destinations to provider names.

A client configuration can use:

```yaml
node:
  name: client-node
  transport: tailscale

tailscale:
  state_dir: ~/.tunny
  auth_key: ${TS_AUTHKEY}

providers:
  japan: 100.64.0.10:7070

routes:
  example.com: japan
```

The provider must run a compatible Tunny configuration with `transport: tailscale`, its own Tailscale hostname, and a provider listener. Use an auth key only through an environment variable or another secret-management mechanism.

This document describes the transport configuration present in the code. It does not claim that the repository provisions a Tailscale network or manages exit-node policy.
