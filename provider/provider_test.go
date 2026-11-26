package provider

import (
	"bufio"
	"context"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Aldiwildan77/tunny/observability/observabilitytest"
)

func TestMain(m *testing.M) {
	observabilitytest.Install()
	os.Exit(m.Run())
}

func roundTrip(t *testing.T, request string) string {
	t.Helper()

	client, server := net.Pipe()
	p := &provider{}
	done := make(chan struct{})
	go func() {
		p.handle(context.Background(), server)
		close(done)
	}()

	_, err := client.Write([]byte(request + "\n"))
	require.NoError(t, err)
	response, err := bufio.NewReader(client).ReadString('\n')
	require.NoError(t, err)
	_ = client.Close()
	<-done
	return strings.TrimSpace(response)
}

func TestHandleRecordsRequestsAndTargetDials(t *testing.T) {
	target, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() {
		for {
			conn, err := target.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	targetAddress := target.Addr().String()

	closed, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	closedAddress := closed.Addr().String()
	require.NoError(t, closed.Close())

	assert.Equal(t, "PONG", roundTrip(t, "PING"))
	assert.Equal(t, "ERR invalid request", roundTrip(t, "HELLO"))
	assert.Equal(t, "OK", roundTrip(t, "CONNECT "+targetAddress))
	assert.True(t, strings.HasPrefix(roundTrip(t, "CONNECT "+closedAddress), "ERR "))
	require.NoError(t, target.Close())

	assert.Equal(t, map[string]float64{
		"tunny.request.type=ping":    1,
		"tunny.request.type=invalid": 1,
		"tunny.request.type=connect": 2,
	}, observabilitytest.Points(t, "tunny.provider.requests"))
	assert.Equal(t, map[string]float64{
		"tunny.outcome=success":                  1,
		"error.type=refused,tunny.outcome=error": 1,
	}, observabilitytest.Points(t, "tunny.provider.target.dial.duration"))
}
