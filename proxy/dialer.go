package proxy

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/Aldiwildan77/tunny/health"
	"github.com/Aldiwildan77/tunny/node"
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

	providers, ok := d.Routes.MatchCandidates(strings.ToLower(host))
	if !ok {
		var dialer net.Dialer
		return dialer.DialContext(ctx, network, address)
	}

	if d.Health != nil {
		providers = d.Health.Candidates(providers)
	}
	var lastErr error
	for _, provider := range providers {
		providerAddress, exists := d.Providers[provider]
		if !exists {
			lastErr = fmt.Errorf("provider not found: %s", provider)
			continue
		}

		log.Printf("Routing %s -> %s\n", address, provider)
		conn, err := d.Node.Dial(ctx, network, providerAddress)
		if err == nil {
			_, err = fmt.Fprintf(conn, "CONNECT %s\n", address)
		}
		var reader *bufio.Reader
		if err == nil {
			reader = bufio.NewReader(conn)
			var response string
			response, err = reader.ReadString('\n')
			if err == nil && strings.TrimSpace(response) != "OK" {
				err = fmt.Errorf("provider error: %s", strings.TrimSpace(response))
			}
		}
		if err != nil {
			if conn != nil {
				_ = conn.Close()
			}
			lastErr = err
			if d.Health != nil {
				d.Health.RecordFailure(provider, err)
			}
			continue
		}
		if d.Health != nil {
			d.Health.RecordSuccess(provider)
		}
		log.Printf("Connected to %s via provider %s\n", address, provider)
		return &bufferedConn{Conn: conn, reader: reader}, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no healthy providers available")
	}
	return nil, &net.OpError{Op: "dial", Net: network, Err: lastErr}
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}
