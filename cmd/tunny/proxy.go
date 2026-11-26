package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Aldiwildan77/tunny/config"
	"github.com/Aldiwildan77/tunny/node"
	"github.com/Aldiwildan77/tunny/observability"
	"github.com/Aldiwildan77/tunny/proxy"
	"github.com/Aldiwildan77/tunny/route"

	"github.com/spf13/cobra"
)

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Start SOCKS5 proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		var obs *observability.Provider

		cfg, err := config.Load(configPath)
		if err != nil {
			return err
		}

		// Daemonize
		isParentProcess, cleanup, err := daemonize(daemonMode, daemonPID, daemonLog)
		if err != nil {
			return err
		}

		if isParentProcess {
			return nil
		}

		defer cleanup()

		// Standard Run

		ctx, stop := signal.NotifyContext(
			context.Background(),
			os.Interrupt,
			syscall.SIGTERM,
		)
		defer stop()

		if cfg.Observability.Enabled {
			obs, err = observability.New(ctx, observability.Config{
				Daemon:     ModeProxy.String(),
				Version:    version,
				InstanceID: cfg.Node.Name,
				Endpoint:   cfg.Observability.Endpoint,
				Insecure:   cfg.Observability.Insecure,
			})
			if err != nil {
				return err
			}

			defer func() {
				if err := obs.Shutdown(context.Background()); err != nil {
					log.Printf("observability shutdown: %v", err)
				}
			}()
		}

		n, err := node.New(
			cfg.Node,
			cfg.Tailscale,
		)
		if err != nil {
			return err
		}

		routes := route.NewWithPolicies(cfg.Routes, routePolicies(cfg))
		healthManager := newHealthManager(cfg, n)
		healthManager.Start(ctx)
		if err := startControlPlane(ctx, cfg, routes, healthManager, ModeProxy.String()); err != nil {
			return err
		}

		p := proxy.New(
			cfg.Proxy.Listen,
			n,
			routes,
			cfg.Providers,
			healthManager,
		)

		return p.Run(ctx)
	},
}
