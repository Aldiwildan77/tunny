package gateway

import (
	"bufio"
	"context"
	"log"
	"net"
	"net/http"
	"time"
)

type Dialer interface {
	Dial(context.Context, string, string) (net.Conn, error)
}

// HTTPProxy is an HTTP forward proxy. It does not terminate TLS.
type HTTPProxy struct {
	Listen string
	Dialer Dialer
}

func NewHTTPProxy(listen string, dialer Dialer) *HTTPProxy {
	return &HTTPProxy{Listen: listen, Dialer: dialer}
}

func (p *HTTPProxy) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", p.Listen)
	if err != nil {
		return err
	}
	defer listener.Close()

	server := &http.Server{Handler: http.HandlerFunc(p.handle)}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	log.Printf("HTTP proxy listening on %s", p.Listen)
	err = server.Serve(listener)
	if ctx.Err() != nil || err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (p *HTTPProxy) handle(writer http.ResponseWriter, request *http.Request) {
	destination, err := requestDestination(request)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	if request.Method == http.MethodConnect {
		p.handleConnect(writer, request, destination)
		return
	}

	p.handleRequest(writer, request, destination)
}

func (p *HTTPProxy) handleConnect(writer http.ResponseWriter, request *http.Request, destination string) {
	upstream, err := p.Dialer.Dial(request.Context(), "tcp", destination)
	if err != nil {
		http.Error(writer, "unable to connect to destination", http.StatusBadGateway)
		return
	}
	defer upstream.Close()

	hijacker, ok := writer.(http.Hijacker)
	if !ok {
		http.Error(writer, "connection hijacking is not supported", http.StatusInternalServerError)
		return
	}

	client, clientBuffer, err := hijacker.Hijack()
	if err != nil {
		return
	}
	defer client.Close()

	if _, err := clientBuffer.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
		return
	}
	if err := clientBuffer.Flush(); err != nil {
		return
	}

	copyBidirectional(client, upstream)
}

func (p *HTTPProxy) handleRequest(writer http.ResponseWriter, request *http.Request, destination string) {
	upstream, err := p.Dialer.Dial(request.Context(), "tcp", destination)
	if err != nil {
		http.Error(writer, "unable to connect to destination", http.StatusBadGateway)
		return
	}
	defer upstream.Close()

	request.RequestURI = ""
	request.Header.Set("Connection", "close")
	if err := request.Write(upstream); err != nil {
		http.Error(writer, "unable to forward request", http.StatusBadGateway)
		return
	}

	response, err := http.ReadResponse(bufio.NewReader(upstream), request)
	if err != nil {
		http.Error(writer, "invalid response from destination", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	response.Header.Set("Connection", "close")
	if err := response.Write(writer); err != nil {
		log.Printf("HTTP proxy response error: %v", err)
	}
}
