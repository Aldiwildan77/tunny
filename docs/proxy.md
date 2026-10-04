---
layout: default
title: Proxy mode
---

# Proxy mode

Proxy mode exposes a SOCKS5 listener for application traffic.

```bash
go run ./cmd/tunny proxy -c example/proxy-provider/client.yaml
```

The default listener is `127.0.0.1:1080`.

## Routing behavior

The SOCKS5 dialer extracts the destination host and port. Exact route matches are sent to the configured provider. Unmatched destinations use a normal local `net.Dialer`.

For matched traffic, the client connects to the provider address through the configured node transport and sends Tunny's internal `CONNECT host:port` request. The provider then connects to the destination.

Test the example with:

```bash
curl --proxy socks5h://127.0.0.1:1080 --connect-timeout 5 --max-time 10 https://example.com
```

`socks5h` leaves hostname resolution with the proxy request. Tunny still resolves configured route hostnames at route-table construction time for IP matching.

## Limitations

The route table supports exact hostname and resolved IP matches only. A missing provider mapping produces a provider-not-found error. The proxy is an application entry point; it does not create a system-wide route.
