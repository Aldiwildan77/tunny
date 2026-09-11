package node

import (
	"context"
	"net"
)

type directNode struct {
	Name     string
	StateDir string
}

func NewDirectNode(name, stateDir string) Node {
	return &directNode{
		Name:     name,
		StateDir: stateDir,
	}
}

func (d *directNode) Start() error {
	return nil
}

func (d *directNode) Close() error {
	return nil
}

func (d *directNode) Dial(ctx context.Context, network string, address string) (net.Conn, error) {
	var dialer net.Dialer
	return dialer.DialContext(ctx, network, address)
}

func (d *directNode) Listen(network string, address string) (net.Listener, error) {
	return net.Listen(network, address)
}
