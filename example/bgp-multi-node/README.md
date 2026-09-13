# BGP for Multi Node setup

Basically, it's just people being nice to their neighbors guys :D

## Topology
```
        DNS (id/jp/sg/us/eu nodes)
        │
        gdnsd
        │
┌───────┼───────┐
▼       ▼       ▼
ID1     ID2     ID3
│       │       │
BGP     BGP     BGP
└───────┼───────┘
        │
        Tunny
```

- gdnsd for selecting the best node
- BGP for exchanging IP prefixes and determining network paths
- tunny for tunneling traffic through a selected provider node
- only ipv4 enabled for now, next use ipv6
- NIC handled by tunny via gvisor + TUN

notes:
- don't forget to calculate the correct subnet!
- bgp/tunny could use gossip protocol (experiment)
- test tunny with chaos engineering (use monkey)

todo:
- add load balancing
- use maglev hashing algorithm behind tunny
- test bouncing connection between jp <> id, if it works, then try to buy eu/us vm
