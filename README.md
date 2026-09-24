# tunny
Route selected internet traffic through trusted nodes using TUN and SOCKS5.

# How it works
Basically it's only proxying the request to provider (node - node) by translatin the host using Tailscale or Direct network

- The client sends traffic to the tunny tunnel or SOCKS5 proxy.
- Tunny checks whether the destination matches a configured route.
- Unmatched traffic uses the normal internet connection.
- Matched traffic is sent to the selected provider node.
- The provider connects to the destination and forwards the response back.
- Providers can connect through Tailscale or a direct network connection.

## Components

### Tunnel

The tunnel creates a local TUN interface. It catches traffic for configured
routes and forwards it to the selected provider.

See the [tunnel and provider example](example/tunnel-provider/README.md).

### Provider

The provider listens for connections from a tunnel or proxy. It connects to
the requested destination and sends data between the client and the internet.

### Proxy

The proxy is a local SOCKS5 server. Applications can use it when they do not
need a system-wide TUN interface.

See the [proxy and provider example](example/proxy-provider/README.md).

## Running as a daemon

See [DAEMON.md](DAEMON.md) for Tunny-managed `--daemon` mode and instructions
for creating `systemd` services on Linux or `launchd` services on macOS.

## Control plane and data plane

### Control plane

The control plane manages Tunny while it is running. It decides where traffic
should go and exposes the settings through gRPC and an HTTP JSON gateway.

It manages:

- Node names and network transport.
- Provider addresses.
- Hostname to provider routes.
- Tunnel, proxy, and provider listening settings.

The control plane API is defined in
[`control-plane/control.proto`](control-plane/control.proto).

The main operations are:

- `GetStatus` checks the running mode and uptime.
- `ListRoutes` lists configured routes.
- `SetRoute` adds or updates a route.
- `DeleteRoute` removes a route.

The gRPC server and JSON gateway are disabled by default. Enable them with:

```yaml
control_plane:
  enabled: true
  listen: 127.0.0.1:7071
  http_listen: 127.0.0.1:7072
```

### Data plane

The data plane carries the actual network traffic after the control plane has
selected a route:

- The tunnel reads packets from the TUN interface.
- The tunnel or proxy opens a connection to the selected provider.
- The provider connects to the destination service.
- Request and response data are copied between the client and destination.

In simple terms, the control plane chooses the path and the data plane moves
the bytes.

### Using the control plane

Start Tunny with the control plane enabled:

```bash
./tunny proxy -c config.yaml
```

The gRPC server listens on `127.0.0.1:7071`. The JSON gateway listens on
`127.0.0.1:7072`.

Use the JSON gateway with `curl`:

```bash
curl http://127.0.0.1:7072/v1/status
curl http://127.0.0.1:7072/v1/routes
curl -X POST http://127.0.0.1:7072/v1/routes \
  -H 'Content-Type: application/json' \
  -d '{"hostname":"example.com","provider":"japan"}'
curl -X DELETE http://127.0.0.1:7072/v1/routes/example.com
```

You can also use gRPC directly with `grpcurl`. The extra import path is
needed because the proto uses Google HTTP annotations:

```bash
GOOGLEAPIS="$(go env GOPATH)/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.16.0/third_party/googleapis"

grpcurl \
  -plaintext \
  -import-path . \
  -import-path "$GOOGLEAPIS" \
  -proto control-plane/control.proto \
  127.0.0.1:7071 \
  tunny.control.v1.ControlPlane/ListRoutes
```

For most use cases, the JSON gateway and `curl` are simpler.

The control plane manages settings. The tunnel, proxy, and provider continue
to carry the actual network traffic.

See the diagrams below

```mermaid
flowchart LR
    subgraph Requestor["Requestor Node"]
        Client["Client<br/>Browser / curl"]
        Proxy["tunny proxy<br/>SOCKS5"]
        Dialer["Dialer<br/>Route Matching"]

        Client -->|"SOCKS5 request"| Proxy
        Proxy --> Dialer
    end

    subgraph TransportNetwork["Transport Network"]
      TS["Direct or encrypted connection"]
    end

    subgraph Provider["Provider Node"]
        ProviderServer["tunny provider<br/>TCP Forwarder"]
        Internet["Internet"]

        ProviderServer -->|"net.Dial()"| Internet
    end

    Dialer -->|"Route matched<br/>CONNECT target"| TS
    TS -->|"Provider connection"| ProviderServer
```

# Flowchart
```mermaid
flowchart LR
    Client["Client<br/>Browser / curl"]
    Proxy["tunny proxy<br/>SOCKS5"]
    Route{"Route match?"}
    Direct["Direct Internet"]
    Provider["Provider<br/>tunny provider"]
    Internet["Internet"]

    Client --> Proxy
    Proxy --> Route

    Route -->|"No"| Direct
    Direct --> Internet

    Route -->|"Yes<br/>domain → provider"| Provider
    Provider -->|"net.Dial()"| Internet

    Proxy -. "Transport network" .-> Provider
```

# Sequence Diagram
```mermaid
sequenceDiagram
    participant C as Client
    participant R as 🇯🇵 Requestor
    participant T as Transport Network
    participant P1 as 🇮🇩 Provider 1
    participant P2 as 🇯🇵 Provider 2
    participant P3 as 🇸🇬 Provider 3
    participant P4 as 🇺🇸 Provider 4
    participant I as Internet

    C->>R: SOCKS5 CONNECT domain:443
    R->>R: Match route
    alt Route to Provider 1
        R->>T: Dial provider 1
        T->>P1: Encrypted connection
        R->>P1: CONNECT domain:443
        P1->>I: Connect to destination
        I-->>P1: Response
        P1-->>R: Response
    else Route to Provider 2
        R->>T: Dial provider 2
        T->>P2: Encrypted connection
        R->>P2: CONNECT domain:443
        P2->>I: Connect to destination
        I-->>P2: Response
        P2-->>R: Response
    else Route to Provider 3
        R->>T: Dial provider 3
        T->>P3: Encrypted connection
        R->>P3: CONNECT domain:443
        P3->>I: Connect to destination
        I-->>P3: Response
        P3-->>R: Response
    else Route to Provider 4
        R->>T: Dial provider 4
        T->>P4: Encrypted connection
        R->>P4: CONNECT domain:443
        P4->>I: Connect to destination
        I-->>P4: Response
        P4-->>R: Response
    end
    R-->>C: Response
```

# Network Topology
```mermaid
flowchart LR
  subgraph Clients["Requestor Nodes"]
    C1["🇯🇵 Client 1<br/>Laptop"]
    C2["🇸🇬 Client 2<br/>Smart TV"]
    C3["🇺🇸 Client 3<br/>Router"]

    T1["Tunny"]
    T2["Tunny"]
    T3["Tunny"]

    C1 --> T1
    C2 --> T2
    C3 --> T3
    end

    subgraph TS["☁️ TRANSPORT NETWORK"]
      T["🔐 Direct or encrypted connection"]
    end

    subgraph Providers["Provider Nodes"]
        P1["🇮🇩 Provider 1"]
        P2["🇯🇵 Provider 2"]
        P3["🇸🇬 Provider 3"]
        P4["🇺🇸 Provider 4"]
    end

    I["🌍 Internet"]

    T1 -->|"Route match"| T
    T2 -->|"Route match"| T
    T3 -->|"Route match"| T
    T -->|"Provider 1"| P1
    T -->|"Provider 2"| P2
    T -->|"Provider 3"| P3
    T -->|"Provider 4"| P4
    P1 -->|"net.Dial()"| I
    P2 -->|"net.Dial()"| I
    P3 -->|"net.Dial()"| I
    P4 -->|"net.Dial()"| I
```

```topojson
{
  "type": "Topology",
  "objects": {
    "routes": {
      "type": "GeometryCollection",
      "geometries": [
        {
          "type": "LineString",
          "properties": {
            "name": "tunny tunnel",
            "from": "Japan",
            "to": "Indonesia"
          },
          "arcs": [0]
        }
      ]
    },
    "nodes": {
      "type": "GeometryCollection",
      "geometries": [
        {
          "type": "Point",
          "properties": {
            "name": "Japan",
            "role": "Requestor"
          },
          "coordinates": [139.6917, 35.6895]
        },
        {
          "type": "Point",
          "properties": {
            "name": "Indonesia",
            "role": "Provider"
          },
          "coordinates": [106.8456, -6.2088]
        }
      ]
    }
  },
  "arcs": [
    [
      [139.6917, 35.6895],
      [135, 28],
      [125, 18],
      [115, 8],
      [106.8456, -6.2088]
    ]
  ]
}
```

## Download and install

The easiest way to get Tunny is to download a release for your operating
system from the [GitHub Releases page](https://github.com/Aldiwildan77/tunny/releases).

After downloading a binary, rename it to `tunny`, make it executable on Unix
systems, and place it somewhere in your `PATH`:

```bash
chmod +x tunny
sudo mv tunny /usr/local/bin/tunny
```

You can also install the latest version with Go:

```bash
go install github.com/Aldiwildan77/tunny/cmd/tunny@latest
```

Or build it from the source code:

```bash
git clone https://github.com/Aldiwildan77/tunny.git
cd tunny
make build
```

The binary is created as `./tunny`. Run `tunny --help` to see the available
commands. Creating a TUN interface may require administrator privileges.
