package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Aldiwildan77/tunny/config"
	"github.com/Aldiwildan77/tunny/node"
	"github.com/Aldiwildan77/tunny/proxy"
	"github.com/Aldiwildan77/tunny/route"

	"github.com/spf13/cobra"
)

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Start SOCKS5 proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configPath)
		if err != nil {
			return err
		}

		ctx, stop := signal.NotifyContext(
			context.Background(),
			os.Interrupt,
			syscall.SIGTERM,
		)
		defer stop()

		n := node.New(
			cfg.Node.Name,
			cfg.Tailscale.StateDir,
			cfg.Tailscale.AuthKey,
		)

		routes := route.New(cfg.Routes)

		p := proxy.New(
			cfg.Proxy.Listen,
			n,
			routes,
			cfg.Providers,
		)

		return p.Run(ctx)
	},
}
