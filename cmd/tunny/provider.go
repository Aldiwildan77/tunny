package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/Aldiwildan77/tunny/config"
	"github.com/Aldiwildan77/tunny/node"
	"github.com/Aldiwildan77/tunny/provider"
)

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "Start TCP provider",
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

		n, err := node.New(
			cfg.Node,
			cfg.Tailscale,
		)
		if err != nil {
			return err
		}

		p := provider.New(
			cfg.Provider.Listen,
			n,
		)

		return p.Run(ctx)
	},
}
