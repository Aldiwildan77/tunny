package node

import (
	"context"
	"fmt"
	"net"

	"github.com/Aldiwildan77/tunny/config"
)

type Node interface {
	Start() error
	Close() error
	Listen(network, address string) (net.Listener, error)
	Dial(ctx context.Context, network, address string) (net.Conn, error)
}

func New(cfg config.NodeConfig, tailscaleConfig config.TailscaleConfig) (Node, error) {
	switch cfg.Transport {
	case "direct":
		return NewDirectNode(cfg.Name, tailscaleConfig.StateDir), nil
	case "tailscale":
		return NewTailscaleNode(cfg.Name, tailscaleConfig.StateDir, tailscaleConfig.AuthKey), nil
	default:
		return nil, fmt.Errorf(
			"unsupported node transport: %s",
			cfg.Transport,
		)
	}
}
