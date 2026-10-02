# Tunnel mode

Tunnel mode accepts system traffic through a TUN device and forwards it with a gVisor userspace network stack.

```bash
sudo go run ./cmd/tunny tunnel -c example/tunnel-provider/client.yaml
```

## Packet flow

1. The platform device supplies IPv4 packets.
2. Tunny injects them into gVisor.
3. gVisor handles IPv4, TCP, and UDP.
4. Tunny derives the destination IP and port.
5. The route-aware dialer connects directly or attempts to use the configured provider.
6. Response packets are written back to the TUN device.

The tunnel installs host routes for resolved route IPs and removes them during cleanup when the platform device implements `RouteManager`.

## Current platform status

- **macOS:** the device implementation includes route management and is the currently supported path for `Tunnel.Run`.
- **Linux:** the WireGuard TUN device exists, but the current implementation does not provide the required route manager, so tunnel startup stops before forwarding.
- **Windows:** the WireGuard TUN device exists, but the current implementation also lacks route management.

The CLI currently constructs `utun` with MTU `1500`; the `tunnel.interface` and `tunnel.mtu` config values are parsed but not applied by the command.

The tunnel stack is IPv4-only. TCP and UDP forwarders are present in gVisor, but the provider listener and `CONNECT` protocol are TCP-only, so end-to-end UDP forwarding through a provider is not currently supported. Real-device tests require platform support and privileges. The tunnel tests create a real TUN device rather than using a fake device.
