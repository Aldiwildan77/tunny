package gateway

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testDialer struct {
	conn net.Conn
}

func (d testDialer) Dial(context.Context, string, string) (net.Conn, error) {
	return d.conn, nil
}

func TestHTTPProxyForwardsRequest(t *testing.T) {
	client, upstream := net.Pipe()
	proxy := &HTTPProxy{Dialer: testDialer{conn: upstream}}

	server := httptest.NewServer(http.HandlerFunc(proxy.handle))
	defer server.Close()

	upstreamDone := make(chan error, 1)
	go func() {
		defer upstream.Close()
		request, err := http.ReadRequest(bufio.NewReader(upstream))
		if err != nil {
			upstreamDone <- err
			return
		}
		if request.URL.Path != "/health" {
			upstreamDone <- fmt.Errorf("unexpected path %q", request.URL.Path)
			return
		}
		_, err = io.WriteString(upstream, "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok\n")
		upstreamDone <- err
	}()

	request, err := http.NewRequest(http.MethodGet, server.URL+"/health", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "example.test:80"
	transport := &http.Transport{DialContext: func(context.Context, string, string) (net.Conn, error) {
		return client, nil
	}}
	response, err := (&http.Client{Transport: transport}).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" {
		t.Fatalf("unexpected response body %q", body)
	}
	if err := <-upstreamDone; err != nil {
		t.Fatal(err)
	}
}
