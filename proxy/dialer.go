package proxy

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/Aldiwildan77/tunny/health"
	"github.com/Aldiwildan77/tunny/node"
	"github.com/Aldiwildan77/tunny/observability"
	"github.com/Aldiwildan77/tunny/route"
)

type Dialer struct {
	Node      node.Node
	Routes    *route.Table
	Providers map[string]string
	Health    *health.Manager
}

func (d *Dialer) Dial(ctx context.Context, network, address string) (net.Conn, error) {
	// if network != "tcp" {
	// 	return nil, fmt.Errorf("unsupported network: %s", network)
	// }

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	match, ok := d.Routes.Lookup(strings.ToLower(host))
	if !ok {
		start := time.Now()
		var dialer net.Dialer
		conn, err := dialer.DialContext(ctx, network, address)
		elapsed := time.Since(start)
		if err != nil {
			errType := observability.DialErrorType(ctx, err)
			recordDirect(ctx, network, elapsed, errType)
			slog.Warn("direct dial failed", "address", address, "error_type", errType, "error", err, "duration", elapsed)
			return nil, err
		}
		recordDirect(ctx, network, elapsed, "")
		slog.Debug("direct dial", "address", address, "duration", elapsed)
		return conn, nil
	}

	start := time.Now()
	providers := match.Providers
	if d.Health != nil {
		providers = d.Health.Candidates(providers)
	}

	slog.Info("route matched",
		"address", address,
		"route", match.Route,
		"candidates", match.Providers,
		"healthy", providers,
	)

	var lastErr error
	lastErrType := errNoHealthyProvider
	for i, provider := range providers {
		attemptStart := time.Now()

		var conn net.Conn
		var reader *bufio.Reader
		var err error

		errType := errProviderNotFound
		providerAddress, exists := d.Providers[provider]
		if !exists {
			err = fmt.Errorf("provider not found: %s", provider)
		}
		if err == nil {
			errType = errProviderUnreachable
			conn, err = d.Node.Dial(ctx, network, providerAddress)
		}
		if err == nil {
			errType = errHandshake
			_, err = fmt.Fprintf(conn, "CONNECT %s\n", address)
		}
		if err == nil {
			reader = bufio.NewReader(conn)
			var response string
			response, err = reader.ReadString('\n')
			if err == nil && strings.TrimSpace(response) != "OK" {
				errType = errProviderRejected
				err = fmt.Errorf("provider error: %s", strings.TrimSpace(response))
			}
		}

		attemptElapsed := time.Since(attemptStart)
		if err != nil {
			if conn != nil {
				_ = conn.Close()
			}
			if d.Health != nil && exists {
				d.Health.RecordFailure(provider, err)
			}

			lastErr, lastErrType = err, classifyError(ctx, err, errType)
			recordAttempt(ctx, network, match.Route, provider, attemptElapsed, lastErrType)
			slog.Warn("route provider attempt failed",
				"address", address,
				"route", match.Route,
				"provider", provider,
				"attempt", i+1,
				"error_type", lastErrType,
				"error", err,
				"duration", attemptElapsed,
			)
			continue
		}

		if d.Health != nil {
			d.Health.RecordSuccess(provider)
		}

		elapsed := time.Since(start)
		recordAttempt(ctx, network, match.Route, provider, attemptElapsed, "")
		recordRequest(ctx, network, match.Route, provider, elapsed, "")
		slog.Info("route connected",
			"address", address,
			"route", match.Route,
			"provider", provider,
			"attempt", i+1,
			"duration", elapsed,
		)
		return &bufferedConn{Conn: conn, reader: reader}, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no healthy providers available")
	}

	elapsed := time.Since(start)
	recordRequest(ctx, network, match.Route, "", elapsed, lastErrType)
	slog.Error("route failed",
		"address", address,
		"route", match.Route,
		"attempts", len(providers),
		"error_type", lastErrType,
		"error", lastErr,
		"duration", elapsed,
	)
	return nil, &net.OpError{Op: "dial", Net: network, Err: lastErr}
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}
