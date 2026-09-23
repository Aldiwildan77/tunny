package provider

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log"
	"net"
	"strings"
	"sync/atomic"

	"github.com/Aldiwildan77/tunny/node"
)

var activeHandlers atomic.Int64

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

		log.Printf("accepted connection from %s", conn.RemoteAddr())
		go p.handle(ctx, conn)
	}
}

func (p *provider) handle(ctx context.Context, conn net.Conn) {
	n := activeHandlers.Add(1)
	log.Printf("handler START active=%d remote=%s", n, conn.RemoteAddr())

	defer func() {
		n := activeHandlers.Add(-1)
		log.Printf("handler END active=%d remote=%s", n, conn.RemoteAddr())
		defer conn.Close()
	}()

	reader := bufio.NewReader(conn)

	request, err := reader.ReadString('\n')
	if err != nil {
		if err != io.EOF {
			log.Printf("read request error: %v\n", err)
			return
		}

		log.Printf("ReadString error: %v\n", err)
		return
	}

	log.Printf("raw request: %q\n", request)

	request = strings.TrimSpace(request)

	log.Printf("parsed request: %q\n", request)

	const prefix = "CONNECT "

	if !strings.HasPrefix(request, prefix) {
		log.Printf("invalid request: %q\n", request)
		_, _ = conn.Write([]byte("ERR invalid request\n"))
		return
	}

	dst := strings.TrimSpace(strings.TrimPrefix(request, prefix))
	log.Printf("destination: %q\n", dst)

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

	log.Printf("Connected to %s\n", dst)

	defer target.Close()

	if _, err := conn.Write([]byte("OK\n")); err != nil {
		log.Printf("write OK error: %v\n", err)
		return
	}

	errCh := make(chan error, 2)

	go func() {
		_, err := io.Copy(target, reader)
		if err != nil {
			log.Printf("copy to target error: %v\n", err)
		}
		errCh <- err
	}()

	go func() {
		_, err := io.Copy(conn, target)
		if err != nil {
			log.Printf("copy to client error: %v\n", err)
		}
		errCh <- err
	}()

	select {
	case err = <-errCh:
	case <-ctx.Done():
		err = ctx.Err()
	}

	_ = conn.Close()
	_ = target.Close()

	if err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, context.Canceled) {
		log.Printf("proxy copy error: %v\n", err)
	}

	if copyErr := <-errCh; copyErr != nil && !errors.Is(copyErr, net.ErrClosed) {
		log.Printf("proxy copy error: %v\n", copyErr)
	}
}

func (p *provider) GetListenAddress() string {
	return p.ListenAddress
}
