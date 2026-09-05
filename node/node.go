package node

import (
	"context"
	"net"

	"tailscale.com/tsnet"
)

type Node interface {
	Start() error
	Close() error
	Listen(network, address string) (net.Listener, error)
	Dial(ctx context.Context, network, address string) (net.Conn, error)
}

type node struct {
	Server *tsnet.Server
}

func New(name, stateDir, authKey string) Node {
	return &node{
		Server: &tsnet.Server{
			Hostname: name,
			Dir:      stateDir,
			AuthKey:  authKey,
		},
	}
}

// Optional, any Dial/Listen will also start the server if it is not already running.
func (n *node) Start() error {
	return n.Server.Start()
}

func (n *node) Close() error {
	return n.Server.Close()
}

func (n *node) Listen(network, address string) (net.Listener, error) {
	return n.Server.Listen(network, address)
}

func (n *node) Dial(ctx context.Context, network, address string) (net.Conn, error) {
	return n.Server.Dial(ctx, network, address)
}
