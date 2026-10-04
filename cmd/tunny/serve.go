package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/Aldiwildan77/tunny/config"
	"github.com/Aldiwildan77/tunny/gateway"
	"github.com/Aldiwildan77/tunny/node"
	"github.com/Aldiwildan77/tunny/proxy"
	"github.com/Aldiwildan77/tunny/route"
	"github.com/Aldiwildan77/tunny/tunnel"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve configured network ingress modes",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configPath)
		if err != nil {
			return err
		}

		isParentProcess, cleanup, err := daemonize(daemonMode, daemonPID, daemonLog)
		if err != nil {
			return err
		}
		if isParentProcess {
			return nil
		}
		defer cleanup()

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		n, err := node.New(cfg.Node, cfg.Tailscale)
		if err != nil {
			return err
		}
		defer n.Close()

		routes := route.New(cfg.Routes)
		if err := startControlPlane(ctx, cfg, routes, ModeServe.String()); err != nil {
			return err
		}

		dialer := &proxy.Dialer{Node: n, Routes: routes, Providers: cfg.Providers}
		ingresses := make([]gateway.Ingress, 0, 3)

		if cfg.Proxy.HTTPListen != "" {
			ingresses = append(ingresses, gateway.NewHTTPProxy(cfg.Proxy.HTTPListen, dialer))
		}

		if cfg.Proxy.Listen != "" {
			ingresses = append(ingresses, gateway.NewSOCKS5(cfg.Proxy.Listen, dialer))
		}

		if cfg.Tunnel.Enabled {
			device, err := tunnel.NewDevice(cfg.Tunnel.Interface, cfg.Tunnel.MTU)
			if err != nil {
				return err
			}

			ingresses = append(ingresses, tunnel.New(device, dialer, routes))
		}

		return gateway.New(ingresses...).Run(ctx)
	},
}
