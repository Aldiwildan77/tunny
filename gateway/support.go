package gateway

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

func requestDestination(request *http.Request) (string, error) {
	destination := request.Host
	if request.Method != http.MethodConnect {
		destination = request.URL.Host
	}
	if destination == "" {
		return "", fmt.Errorf("request has no destination")
	}
	if _, _, err := net.SplitHostPort(destination); err != nil {
		if strings.Contains(err.Error(), "missing port") {
			port := "80"
			if request.URL != nil && strings.EqualFold(request.URL.Scheme, "https") {
				port = "443"
			}
			destination = net.JoinHostPort(destination, port)
		} else {
			return "", fmt.Errorf("invalid destination: %w", err)
		}
	}
	return destination, nil
}

func copyBidirectional(left, right net.Conn) {
	result := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(left, right)
		if tcp, ok := left.(*net.TCPConn); ok {
			_ = tcp.CloseWrite()
		}
		result <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(right, left)
		if tcp, ok := right.(*net.TCPConn); ok {
			_ = tcp.CloseWrite()
		}
		result <- struct{}{}
	}()
	<-result
	_ = left.Close()
	_ = right.Close()
}
