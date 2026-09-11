package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/Aldiwildan77/tunny/config"
	"github.com/Aldiwildan77/tunny/node"
	"github.com/Aldiwildan77/tunny/proxy"
	"github.com/Aldiwildan77/tunny/route"
	"github.com/Aldiwildan77/tunny/tunnel"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Start transparent network tunnel",
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

		routes := route.New(cfg.Routes)

		log.Printf("config routes: %+v", cfg.Routes)
		log.Printf("route IPs: %+v", routes.GetIPs())

		dialer := &proxy.Dialer{
			Node:      n,
			Routes:    routes,
			Providers: cfg.Providers,
		}

		ips := make([]net.IP, 0)

		for _, address := range routes.GetNetIPs() {
			ip := net.ParseIP(address.String())
			if ip == nil || ip.To4() == nil {
				continue
			}

			ips = append(ips, ip)
		}

		log.Printf("resolved tunnel IPs: %v", ips)

		device, err := tunnel.NewDevice("utun", 1500)
		if err != nil {
			return err
		}

		t := tunnel.New(
			device,
			dialer,
			routes,
		)

		return t.Run(ctx)
	},
}
