---
layout: default
title: Single-provider example
---

# Single provider

This is the smallest working example: one local provider and one local SOCKS5 client.

## Start the provider

```bash
go run ./cmd/tunny provider -c example/proxy-provider/provider.yaml
```

## Start the proxy

In another terminal:

```bash
go run ./cmd/tunny proxy -c example/proxy-provider/client.yaml
```

The provider listens on `127.0.0.1:7070`. The proxy listens on `127.0.0.1:1080` and maps `example.com` to the provider.

## Test

```bash
curl --proxy socks5h://127.0.0.1:1080 --connect-timeout 5 --max-time 10 https://example.com
```

The full example README is [here](https://github.com/Aldiwildan77/tunny/blob/master/example/proxy-provider/README.md).
