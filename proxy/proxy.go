package proxy

import (
	"context"
	"log"
	"net"

	"github.com/Aldiwildan77/tunny/node"
	"github.com/Aldiwildan77/tunny/route"
	"github.com/armon/go-socks5"
)

type Proxy interface {
	Run(ctx context.Context) error
}

type proxy struct {
	Listen    string
	Node      node.Node
	Routes    *route.Table
	Providers map[string]string
}

func New(listen string, node node.Node, routes *route.Table, providers map[string]string) Proxy {
	return &proxy{
		Listen:    listen,
		Node:      node,
		Routes:    routes,
		Providers: providers,
	}
}

func (p *proxy) Run(ctx context.Context) error {
	dialer := &Dialer{
		Node:      p.Node,
		Routes:    p.Routes,
		Providers: p.Providers,
	}

	conf := &socks5.Config{
		Dial: dialer.Dial,
	}

	server, err := socks5.New(conf)
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", p.Listen)
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Printf("SOCKS5 proxy listening on %s\n", p.Listen)

	go func() {
		<-ctx.Done()
		log.Println("Shutting down proxy...")
		listener.Close()
	}()

	err = server.Serve(listener)
	if err != nil && ctx.Err() != nil {
		return nil
	}

	return err
}
