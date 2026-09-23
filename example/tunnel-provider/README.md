# Tunnel and provider

This example runs a local Tunny tunnel and a local provider. Traffic for
`example.com` is sent through the provider. Other traffic uses the normal
network connection.

## Start the provider

From the repository root:

```bash
go run ./cmd/tunny provider -c example/tunnel-provider/provider.yaml
```

## Start the tunnel

In another terminal:

```bash
sudo go run ./cmd/tunny tunnel -c example/tunnel-provider/client.yaml
```

The tunnel creates a TUN interface and may ask for administrator privileges.

## Test it

In a third terminal:

```bash
curl --connect-timeout 5 --max-time 10 https://example.com
```
