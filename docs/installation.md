---
layout: default
title: Installation
---

# Installation

Tunny is distributed as release archives and can also be built directly from
source. Choose the installation method that matches how you operate the node.

## Release binary

Download the archive for your operating system and architecture from the
[GitHub Releases page](https://github.com/Aldiwildan77/tunny/releases). The
release includes the `tunny` binary and an example `config.yaml`.

For macOS or Linux:

```bash
VERSION="0.4.0"
OS="darwin"       # linux or darwin
ARCH="arm64"      # amd64 or arm64

curl -LO "https://github.com/Aldiwildan77/tunny/releases/download/v${VERSION}/tunny_${VERSION}_${OS}_${ARCH}.tar.gz"
tar -xzf "tunny_${VERSION}_${OS}_${ARCH}.tar.gz"
sudo install tunny /usr/local/bin/tunny
```

Windows releases are published as ZIP archives. Extract `tunny.exe` and add
its directory to `PATH`.

Check the installation:

```bash
tunny --help
tunny --config config.yaml serve
```

## Homebrew

The repository publishes a Homebrew cask with tagged releases. Once the cask
is available in the project tap, install it with:

```bash
brew install --cask Aldiwildan77/tunny/tunny
```

The generated cask definition is maintained at
[`Casks/tunny.rb`](https://github.com/Aldiwildan77/tunny/blob/master/Casks/tunny.rb).

## Build from source

Source builds require the Go version declared in [`go.mod`](https://github.com/Aldiwildan77/tunny/blob/master/go.mod):

```bash
git clone https://github.com/Aldiwildan77/tunny.git
cd tunny
go build -o tunny ./cmd/tunny
sudo install tunny /usr/local/bin/tunny
```

Run the repository checks before installing a local build:

```bash
make check
```

The TUN tests create real devices and may require platform support and
elevated privileges. The proxy, provider, and serve configuration can be
tested without enabling the TUN interface.

## First configuration

Copy the example configuration and set the node identity and transport:

```bash
cp .example.config.yaml config.yaml
```

For a minimal proxy-only setup, disable the TUN ingress:

```yaml
node:
  name: client
  transport: direct

tunnel:
  enabled: false

proxy:
  listen: 127.0.0.1:1080
```

A provider node needs a reachable listener:

```yaml
node:
  name: provider-1
  transport: direct

provider:
  listen: ":7070"
```

See [Serve mode](serve.html), [Providers](providers.html), and the
[Single-provider example](examples/single-provider.html) for a complete
client/provider setup.

## Permissions and access

Keep proxy and control-plane listeners on loopback unless an access-control
layer protects them. Transparent TUN mode may require `sudo` or equivalent
network privileges depending on the operating system. Never expose an
unauthenticated proxy to an untrusted network.
