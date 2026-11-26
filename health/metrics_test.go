package health

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Aldiwildan77/tunny/observability/observabilitytest"
)

func TestMain(m *testing.M) {
	observabilitytest.Install()
	os.Exit(m.Run())
}

type pingNode struct {
	reply map[string]string
}

func (n pingNode) Start() error { return nil }
func (n pingNode) Close() error { return nil }
func (n pingNode) Listen(string, string) (net.Listener, error) {
	return nil, errors.New("not supported")
}
func (n pingNode) Dial(_ context.Context, _ string, address string) (net.Conn, error) {
	reply, ok := n.reply[address]
	if !ok {
		return nil, errors.New("connection refused")
	}
	client, server := net.Pipe()
	go func() {
		defer server.Close()
		_, _ = server.Read(make([]byte, 16))
		_, _ = server.Write([]byte(reply + "\n"))
	}()
	return client, nil
}

func TestHealthMetrics(t *testing.T) {
	manager := New(pingNode{reply: map[string]string{
		"up:7070":    "PONG",
		"weird:7070": "NOPE",
	}}, map[string]string{
		"metrics-up":    "up:7070",
		"metrics-weird": "weird:7070",
		"metrics-down":  "down:7070",
	}, Config{FailureThreshold: 1, Timeout: time.Second})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager.registerMetrics(ctx)
	manager.checkAll(ctx)

	checks := observabilitytest.Points(t, "tunny.health.check.duration")
	assert.Equal(t, float64(1), checks["tunny.outcome=success,tunny.provider=metrics-up"])
	assert.Equal(t, float64(1), checks["error.type=unexpected_response,tunny.outcome=error,tunny.provider=metrics-weird"])
	assert.Equal(t, float64(1), checks["error.type=provider_unreachable,tunny.outcome=error,tunny.provider=metrics-down"])

	transitions := observabilitytest.Points(t, "tunny.health.transitions")
	assert.Equal(t, float64(1), transitions["tunny.health.state=unhealthy,tunny.provider=metrics-down"])
	assert.Equal(t, float64(1), transitions["tunny.health.state=unhealthy,tunny.provider=metrics-weird"])

	healthy := observabilitytest.Points(t, "tunny.health.healthy")
	assert.Equal(t, float64(1), healthy["tunny.provider=metrics-up"])
	assert.Equal(t, float64(0), healthy["tunny.provider=metrics-down"])
	require.Contains(t, healthy, "tunny.provider=metrics-weird")
}
