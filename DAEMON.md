# Running Tunny as a Daemon

Tunny supports two ways to run a long-lived process:

1. Tunny-managed daemon mode with `--daemon`.
2. Operating-system supervision using `systemd` on Linux or `launchd` on macOS.

Choose one method for a given process. Do not combine `--daemon` with `systemd` or `launchd`.

## Tunny-managed daemon mode

Build a stable executable first:

```bash
go build -o tunny ./cmd/tunny
```

Start a proxy in the background:

```bash
./tunny \
  --daemon \
  --pid /tmp/tunny-proxy.pid \
  --log /tmp/tunny-proxy.log \
  --config config.yaml \
  proxy
```

Start a provider:

```bash
./tunny \
  --daemon \
  --pid /tmp/tunny-provider.pid \
  --log /tmp/tunny-provider.log \
  --config config.yaml \
  provider
```

Check status:

```bash
./tunny --pid /tmp/tunny-proxy.pid daemon status
```

Stop the process:

```bash
./tunny --pid /tmp/tunny-proxy.pid daemon stop
```

Inspect logs:

```bash
tail -f /tmp/tunny-proxy.log
```

Global flags must appear before the subcommand:

```bash
./tunny --pid /tmp/tunny-proxy.pid daemon status
```

Use an absolute config path when launching Tunny from scripts or another working directory:

```bash
./tunny \
  --daemon \
  --pid /tmp/tunny-proxy.pid \
  --log /tmp/tunny-proxy.log \
  --config /etc/tunny/config.yaml \
  proxy
```

The default paths are `/var/run/tunny.pid` and `/var/log/tunny.log`. These locations generally require elevated privileges. For an unprivileged user, provide writable paths explicitly.

## Linux with systemd

Create the service yourself at `/etc/systemd/system/tunny.service`:

```ini
[Unit]
Description=Tunny network proxy
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/tunny --config /etc/tunny/config.yaml proxy
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Install the binary and configuration, then enable the service:

```bash
sudo install -Dm755 ./tunny /usr/local/bin/tunny
sudo install -Dm644 config.yaml /etc/tunny/config.yaml
sudo systemctl daemon-reload
sudo systemctl enable --now tunny
```

Check status and logs:

```bash
sudo systemctl status tunny
sudo journalctl -u tunny -f
```

Stop, restart, or disable it:

```bash
sudo systemctl stop tunny
sudo systemctl restart tunny
sudo systemctl disable tunny
```

When using systemd, omit `--daemon`. systemd owns the process lifecycle, restart policy, and logs.

## macOS with launchd

Create a system service at `/Library/LaunchDaemons/com.tunny.proxy.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.tunny.proxy</string>

    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/tunny</string>
        <string>--config</string>
        <string>/etc/tunny/config.yaml</string>
        <string>proxy</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>

    <key>StandardOutPath</key>
    <string>/var/log/tunny.log</string>

    <key>StandardErrorPath</key>
    <string>/var/log/tunny-error.log</string>
</dict>
</plist>
```

Install and load it:

```bash
sudo install -Dm755 ./tunny /usr/local/bin/tunny
sudo install -Dm644 config.yaml /etc/tunny/config.yaml
sudo chown root:wheel /Library/LaunchDaemons/com.tunny.proxy.plist
sudo chmod 644 /Library/LaunchDaemons/com.tunny.proxy.plist
sudo launchctl bootstrap system /Library/LaunchDaemons/com.tunny.proxy.plist
sudo launchctl enable system/com.tunny.proxy
```

Check status and logs:

```bash
sudo launchctl print system/com.tunny.proxy
sudo tail -f /var/log/tunny.log /var/log/tunny-error.log
```

Stop and unload it:

```bash
sudo launchctl bootout system/com.tunny.proxy
```

When using launchd, omit `--daemon`. launchd owns the process lifecycle and restart behavior.

## Tunnel mode permissions

Tunnel mode creates a TUN device and changes routes. It may require elevated privileges and platform-specific network permissions:

```bash
sudo ./tunny \
  --daemon \
  --pid /var/run/tunny.pid \
  --log /var/log/tunny.log \
  --config /etc/tunny/config.yaml \
  tunnel
```

For systemd or launchd, put the foreground `tunnel` command in the service definition and let the service manager supervise it.

## Control-plane health check

If the control plane is enabled, check the running process with:

```bash
curl http://127.0.0.1:7072/v1/status
```

A daemon status check confirms that the process exists. The control-plane endpoint confirms that Tunny has started its application services.
