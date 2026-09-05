package provider

import (
	"bufio"
	"context"
	"io"
	"log"
	"net"
	"strings"

	"github.com/Aldiwildan77/tunny/node"
)

type Provider interface {
	Run(ctx context.Context) error
	GetListenAddress() string
}

type provider struct {
	ListenAddress string
	Node          node.Node
}

func New(listenAddress string, node node.Node) Provider {
	return &provider{
		ListenAddress: listenAddress,
		Node:          node,
	}
}

func (p *provider) Run(ctx context.Context) error {
	ln, err := p.Node.Listen("tcp", p.ListenAddress)
	if err != nil {
		return err
	}
	defer ln.Close()

	log.Println("Server is running on", p.ListenAddress)

	go func() {
		<-ctx.Done()
		log.Println("Shutting down server...")
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			log.Printf("accept error: %v\n", err)
			continue
		}

		go p.handle(ctx, conn)
	}
}

func (p *provider) handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	request, err := reader.ReadString('\n')
	if err != nil {
		if err != io.EOF {
			log.Printf("read request error: %v\n", err)
		}
		return
	}

	request = strings.TrimSpace(request)

	const prefix = "CONNECT "

	if !strings.HasPrefix(request, prefix) {
		_, _ = conn.Write([]byte("ERR invalid request\n"))
		return
	}

	dst := strings.TrimSpace(strings.TrimPrefix(request, prefix))

	if dst == "" {
		_, _ = conn.Write([]byte("ERR missing destination\n"))
		return
	}

	log.Printf("Connecting to %s...", dst)

	target, err := net.Dial("tcp", dst)
	if err != nil {
		log.Printf("dial target error: %v", err)

		_, _ = conn.Write(
			[]byte("ERR " + err.Error() + "\n"),
		)

		return
	}

	defer target.Close()

	if _, err := conn.Write([]byte("OK\n")); err != nil {
		log.Printf("write OK error: %v", err)
		return
	}

	log.Printf("Connected to %s", dst)

	go func() {
		if _, err := io.Copy(target, reader); err != nil {
			log.Printf("copy to target error: %v", err)
		}
	}()

	if _, err := io.Copy(conn, target); err != nil {
		log.Printf("copy to client error: %v", err)
	}
}

func (p *provider) GetListenAddress() string {
	return p.ListenAddress
}
