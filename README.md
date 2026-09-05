# tunny
Location-aware network tunneling for routing traffic through trusted nodes.

# How it works
Basically it's only proxying the request to provider (node - node) by translatin the host using Tailscale network

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

    subgraph Tailscale["Tailscale Tailnet"]
        TS["Encrypted connection"]
    end

    subgraph Provider["Provider Node"]
        ProviderServer["tunny provider<br/>TCP Forwarder"]
        Internet["Internet"]

        ProviderServer -->|"net.Dial()"| Internet
    end

    Dialer -->|"Route matched<br/>CONNECT target"| TS
    TS -->|"tsnet"| ProviderServer
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

    Proxy -. "tsnet" .-> Provider
```

# Sequence Diagram
```mermaid
sequenceDiagram
    participant C as Client
    participant R as 🇯🇵 Requestor
    participant T as Tailscale
    participant P as 🇮🇩 Provider
    participant I as Internet

    C->>R: SOCKS5 CONNECT domain:443
    R->>R: Match route
    R->>T: Dial indonesia:7070
    T->>P: tsnet connection
    R->>P: CONNECT domain:443
    P->>I: net.Dial(domain:443)
    I-->>P: Response
    P-->>R: Response
    R-->>C: Response
```

# Network Topology
```mermaid
flowchart LR
    subgraph JP["🇯🇵 JAPAN"]
        B["🌐 Browser"]
        R["tunny proxy<br/>SOCKS5"]
        D["Route Matcher"]

        B -->|"SOCKS5"| R
        R --> D
    end

    subgraph TS["☁️ TAILSCALE TAILNET"]
        T["🔐 tsnet<br/>Encrypted tunnel"]
    end

    subgraph ID["🇮🇩 INDONESIA"]
        P["tunny provider<br/>TCP Forwarder"]
        I["🌍 Internet"]

        P -->|"net.Dial()"| I
    end

    D -->|"rakuten.co.jp → indonesia"| T
    T -->|"TCP"| P

    classDef requestor fill:#e8f1ff,stroke:#3b82f6,stroke-width:2px
    classDef provider fill:#eaf8ef,stroke:#22c55e,stroke-width:2px
    classDef tunnel fill:#f5edff,stroke:#a855f7,stroke-width:2px
    classDef internet fill:#f5f5f5,stroke:#737373,stroke-width:2px

    class B,R,D requestor
    class P provider
    class T tunnel
    class I internet
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