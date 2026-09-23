package main

import (
	"context"
	"log"

	"github.com/Aldiwildan77/tunny/config"
	controlplane "github.com/Aldiwildan77/tunny/control-plane"
	"github.com/Aldiwildan77/tunny/route"
)

func startControlPlane(
	ctx context.Context,
	cfg *config.Config,
	routes *route.Table,
	mode string,
) error {
	if !cfg.Control.Enabled {
		return nil
	}

	server, err := controlplane.NewServer(
		routes,
		mode,
		cfg.Node.Name,
		cfg.Node.Transport,
	)
	if err != nil {
		return err
	}

	go func() {
		if err := server.Serve(ctx, cfg.Control.Listen, cfg.Control.HTTPListen); err != nil && ctx.Err() == nil {
			log.Printf("control plane stopped: %v", err)
		}
	}()

	log.Printf("control plane gRPC listening on %s", cfg.Control.Listen)
	log.Printf("control plane HTTP listening on %s", cfg.Control.HTTPListen)
	return nil
}
