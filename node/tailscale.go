package node

import (
	"context"
	"net"

	"tailscale.com/tsnet"
)

type tailscaleNode struct {
	Server *tsnet.Server
}

func NewTailscaleNode(name, stateDir, authKey string) Node {
	return &tailscaleNode{
		Server: &tsnet.Server{
			Hostname: name,
			Dir:      stateDir,
			AuthKey:  authKey,
		},
	}
}

// Optional, any Dial/Listen will also start the server if it is not already running.
func (n *tailscaleNode) Start() error {
	return n.Server.Start()
}

func (n *tailscaleNode) Close() error {
	return n.Server.Close()
}

func (n *tailscaleNode) Listen(network, address string) (net.Listener, error) {
	return n.Server.Listen(network, address)
}

func (n *tailscaleNode) Dial(ctx context.Context, network, address string) (net.Conn, error) {
	return n.Server.Dial(ctx, network, address)
}
