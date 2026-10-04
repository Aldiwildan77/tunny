package gateway

import (
	"context"
	"errors"
	"sync"

	"github.com/Aldiwildan77/tunny/proxy"
)

// Ingress is a protocol adapter that accepts client traffic.
type Ingress interface {
	Run(context.Context) error
}

type SOCKS5 struct {
	proxy proxy.Proxy
}

func NewSOCKS5(listen string, dialer *proxy.Dialer) Ingress {
	return &SOCKS5{proxy: proxy.New(listen, dialer.Node, dialer.Routes, dialer.Providers)}
}

func (s *SOCKS5) Run(ctx context.Context) error {
	return s.proxy.Run(ctx)
}

// Gateway runs multiple ingress adapters against one shared routing setup.
type Gateway struct {
	ingresses []Ingress
}

func New(ingresses ...Ingress) *Gateway {
	return &Gateway{ingresses: ingresses}
}

func (g *Gateway) Run(ctx context.Context) error {
	if len(g.ingresses) == 0 {
		return errors.New("gateway has no enabled ingresses")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, len(g.ingresses))
	var wg sync.WaitGroup

	for _, ingress := range g.ingresses {
		wg.Add(1)
		go func(ingress Ingress) {
			defer wg.Done()
			if err := ingress.Run(ctx); err != nil && ctx.Err() == nil {
				errCh <- err
				cancel()
			}
		}(ingress)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case err := <-errCh:
		<-done
		return err
	case <-ctx.Done():
		<-done
		return nil
	case <-done:
		return nil
	}
}
