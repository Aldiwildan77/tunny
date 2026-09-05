package proxy

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/Aldiwildan77/tunny/node"
	"github.com/Aldiwildan77/tunny/route"
)

type Dialer struct {
	Node      node.Node
	Routes    *route.Table
	Providers map[string]string
}

func (d *Dialer) Dial(ctx context.Context, network, address string) (net.Conn, error) {
	// if network != "tcp" {
	// 	return nil, fmt.Errorf("unsupported network: %s", network)
	// }

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	provider, ok := d.Routes.Match(strings.ToLower(host))
	if !ok {
		var dialer net.Dialer
		return dialer.DialContext(ctx, network, address)
	}

	log.Printf("Routing %s -> %s\n", address, provider)

	providerAddress, ok := d.Providers[provider]
	if !ok {
		return nil, &net.OpError{
			Op:  "dial",
			Net: network,
			Err: &net.AddrError{
				Err:  "provider not found",
				Addr: provider,
			},
		}
	}

	conn, err := d.Node.Dial(ctx, network, providerAddress)
	if err != nil {
		return nil, &net.OpError{
			Op:  "dial",
			Net: network,
			Err: err,
		}
	}

	if _, err := fmt.Fprintf(conn, "CONNECT %s\n", address); err != nil {
		conn.Close()

		return nil, &net.OpError{
			Op:  "write",
			Net: network,
			Err: err,
		}
	}

	reader := bufio.NewReader(conn)

	response, err := reader.ReadString('\n')
	if err != nil {
		conn.Close()

		return nil, &net.OpError{
			Op:  "read",
			Net: network,
			Err: err,
		}
	}

	response = strings.TrimSpace(response)

	if response != "OK" {
		conn.Close()

		log.Printf("Provider %s returned error (!OK): %s\n", provider, response)

		return nil, &net.OpError{
			Op:  "read",
			Net: network,
			Err: fmt.Errorf("provider error: %s", response),
		}
	}

	log.Printf("Connected to %s via provider %s\n", address, provider)

	return &bufferedConn{
		Conn:   conn,
		reader: reader,
	}, nil
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}
