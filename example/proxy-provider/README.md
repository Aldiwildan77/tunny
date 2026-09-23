# Proxy and provider

This example runs a local SOCKS5 proxy and a local provider. Traffic for
`example.com` is sent through the provider. Other traffic uses the normal
network connection.

## Start the provider

From the repository root:

```bash
go run ./cmd/tunny provider -c example/proxy-provider/provider.yaml
```

## Start the SOCKS5 proxy

In another terminal:

```bash
go run ./cmd/tunny proxy -c example/proxy-provider/client.yaml
```

The proxy listens on `127.0.0.1:1080`.

## Test it

In a third terminal:

```bash
curl --proxy socks5h://127.0.0.1:1080 --connect-timeout 5 --max-time 10 https://example.com
```
