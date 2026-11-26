package gateway

import (
	"bufio"
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/Aldiwildan77/tunny/observability"
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
	listener = observability.TrackListener(listener, observability.IngressHTTP)
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
	start := time.Now()
	status := p.serve(writer, request)
	recordHTTPRequest(request.Context(), request.Method, status, time.Since(start))
}

func (p *HTTPProxy) serve(writer http.ResponseWriter, request *http.Request) int {
	destination, err := requestDestination(request)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return http.StatusBadRequest
	}

	if request.Method == http.MethodConnect {
		return p.handleConnect(writer, request, destination)
	}

	return p.handleRequest(writer, request, destination)
}

func (p *HTTPProxy) handleConnect(writer http.ResponseWriter, request *http.Request, destination string) int {
	upstream, err := p.Dialer.Dial(request.Context(), "tcp", destination)
	if err != nil {
		http.Error(writer, "unable to connect to destination", http.StatusBadGateway)
		return http.StatusBadGateway
	}
	defer upstream.Close()

	hijacker, ok := writer.(http.Hijacker)
	if !ok {
		http.Error(writer, "connection hijacking is not supported", http.StatusInternalServerError)
		return http.StatusInternalServerError
	}

	client, clientBuffer, err := hijacker.Hijack()
	if err != nil {
		return http.StatusInternalServerError
	}
	defer client.Close()

	if _, err := clientBuffer.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
		return http.StatusOK
	}
	if err := clientBuffer.Flush(); err != nil {
		return http.StatusOK
	}

	copyBidirectional(client, upstream)
	return http.StatusOK
}

func (p *HTTPProxy) handleRequest(writer http.ResponseWriter, request *http.Request, destination string) int {
	upstream, err := p.Dialer.Dial(request.Context(), "tcp", destination)
	if err != nil {
		http.Error(writer, "unable to connect to destination", http.StatusBadGateway)
		return http.StatusBadGateway
	}
	defer upstream.Close()

	request.RequestURI = ""
	request.Header.Set("Connection", "close")
	if err := request.Write(upstream); err != nil {
		http.Error(writer, "unable to forward request", http.StatusBadGateway)
		return http.StatusBadGateway
	}

	response, err := http.ReadResponse(bufio.NewReader(upstream), request)
	if err != nil {
		http.Error(writer, "invalid response from destination", http.StatusBadGateway)
		return http.StatusBadGateway
	}
	defer response.Body.Close()
	response.Header.Set("Connection", "close")
	if err := response.Write(writer); err != nil {
		log.Printf("HTTP proxy response error: %v", err)
	}
	return response.StatusCode
}
