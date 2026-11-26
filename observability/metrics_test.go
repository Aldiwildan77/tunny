package observability_test

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Aldiwildan77/tunny/observability"
	"github.com/Aldiwildan77/tunny/observability/observabilitytest"
)

func TestMain(m *testing.M) {
	observabilitytest.Install()
	os.Exit(m.Run())
}

func TestTrackConnRecordsLifecycle(t *testing.T) {
	client, server := net.Pipe()
	tracked := observability.TrackConn(server, observability.IngressSOCKS5, "tcp")

	assert.Equal(t, int64(1), observability.ActiveConnections())
	assert.Equal(t, map[string]float64{
		"network.transport=tcp,tunny.ingress=socks5": 1,
	}, observabilitytest.Points(t, "tunny.connection.active"))

	go func() {
		_, _ = client.Write([]byte("hello"))
		_, _ = io.ReadFull(client, make([]byte, 3))
	}()

	buf := make([]byte, 5)
	_, err := io.ReadFull(tracked, buf)
	require.NoError(t, err)
	_, err = tracked.Write([]byte("hey"))
	require.NoError(t, err)

	require.NoError(t, tracked.Close())
	require.NoError(t, tracked.Close())
	_ = client.Close()

	assert.Equal(t, int64(0), observability.ActiveConnections())
	assert.Equal(t, map[string]float64{
		"network.transport=tcp,tunny.ingress=socks5": 0,
	}, observabilitytest.Points(t, "tunny.connection.active"))
	assert.Equal(t, map[string]float64{
		"network.transport=tcp,tunny.ingress=socks5": 1,
	}, observabilitytest.Points(t, "tunny.connection.duration"))
	assert.Equal(t, map[string]float64{
		"network.io.direction=receive,network.transport=tcp,tunny.ingress=socks5":  5,
		"network.io.direction=transmit,network.transport=tcp,tunny.ingress=socks5": 3,
	}, observabilitytest.Points(t, "tunny.connection.io"))
}

func TestDialErrorType(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name string
		ctx  context.Context
		err  error
		want string
	}{
		{name: "canceled context", ctx: canceled, err: errors.New("boom"), want: "canceled"},
		{name: "deadline", ctx: context.Background(), err: context.DeadlineExceeded, want: "timeout"},
		{name: "dns", ctx: context.Background(), err: &net.OpError{Op: "dial", Err: &net.DNSError{Err: "no such host"}}, want: "dns"},
		{name: "refused", ctx: context.Background(), err: &net.OpError{Op: "dial", Err: os.NewSyscallError("connect", syscall.ECONNREFUSED)}, want: "refused"},
		{name: "unreachable", ctx: context.Background(), err: &net.OpError{Op: "dial", Err: os.NewSyscallError("connect", syscall.EHOSTUNREACH)}, want: "unreachable"},
		{name: "other", ctx: context.Background(), err: errors.New("boom"), want: "dial"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, observability.DialErrorType(tt.ctx, tt.err))
		})
	}
}

func TestHTTPMethod(t *testing.T) {
	assert.Equal(t, "GET", observability.HTTPMethod("GET"))
	assert.Equal(t, "_OTHER", observability.HTTPMethod("BREW"))
}
