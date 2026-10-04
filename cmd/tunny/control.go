package main

import (
	"context"
	"log"

	"github.com/Aldiwildan77/tunny/config"
	controlplane "github.com/Aldiwildan77/tunny/control-plane"
	"github.com/Aldiwildan77/tunny/health"
	"github.com/Aldiwildan77/tunny/node"
	"github.com/Aldiwildan77/tunny/route"
)

func startControlPlane(
	ctx context.Context,
	cfg *config.Config,
	routes *route.Table,
	healthManager *health.Manager,
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
		healthManager,
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

func newHealthManager(cfg *config.Config, n node.Node) *health.Manager {
	manager := health.New(n, cfg.Providers, health.Config{
		Enabled:           cfg.Health.Enabled,
		Interval:          cfg.Health.Interval,
		Timeout:           cfg.Health.Timeout,
		FailureThreshold:  cfg.Health.FailureThreshold,
		RecoveryThreshold: cfg.Health.RecoveryThreshold,
	})
	return manager
}

func routePolicies(cfg *config.Config) map[string][]string {
	policies := make(map[string][]string, len(cfg.RoutePolicies))
	for hostname, policy := range cfg.RoutePolicies {
		if policy.Mode == "failover" && len(policy.Providers) > 0 {
			policies[hostname] = append([]string(nil), policy.Providers...)
		}
	}
	return policies
}
