package proxy

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Aldiwildan77/tunny/observability/observabilitytest"
	"github.com/Aldiwildan77/tunny/route"
)

func TestMain(m *testing.M) {
	observabilitytest.Install()
	os.Exit(m.Run())
}

type fakeNode struct {
	dial map[string]func() (net.Conn, error)
}

func (n fakeNode) Start() error { return nil }
func (n fakeNode) Close() error { return nil }
func (n fakeNode) Listen(string, string) (net.Listener, error) {
	return nil, errors.New("not supported")
}
func (n fakeNode) Dial(_ context.Context, _ string, address string) (net.Conn, error) {
	return n.dial[address]()
}

func replying(response string) func() (net.Conn, error) {
	return func() (net.Conn, error) {
		client, server := net.Pipe()
		go func() {
			defer server.Close()
			if _, err := bufio.NewReader(server).ReadString('\n'); err != nil {
				return
			}
			if response == "" {
				return
			}
			_, _ = io.WriteString(server, response+"\n")
			_, _ = io.Copy(io.Discard, server)
		}()
		return client, nil
	}
}

func unreachable() (net.Conn, error) {
	return nil, errors.New("connection refused")
}

func routePoints(t *testing.T, name, route string) map[string]float64 {
	t.Helper()

	points := make(map[string]float64)
	for key, value := range observabilitytest.Points(t, name) {
		if strings.Contains(key, "tunny.route="+route+",") || strings.HasSuffix(key, "tunny.route="+route) {
			points[key] = value
		}
	}
	return points
}

func TestDialRecordsRouteMetrics(t *testing.T) {
	tests := []struct {
		name         string
		host         string
		table        *route.Table
		providers    map[string]string
		dial         map[string]func() (net.Conn, error)
		wantErr      bool
		wantRequests map[string]float64
		wantAttempts map[string]float64
	}{
		{
			name:      "success on first provider",
			host:      "ok.example",
			table:     &route.Table{Routes: map[string]string{"ok.example": "a"}},
			providers: map[string]string{"a": "a:7070"},
			dial:      map[string]func() (net.Conn, error){"a:7070": replying("OK")},
			wantRequests: map[string]float64{
				"network.transport=tcp,tunny.outcome=success,tunny.provider=a,tunny.route=ok.example": 1,
			},
			wantAttempts: map[string]float64{
				"network.transport=tcp,tunny.outcome=success,tunny.provider=a,tunny.route=ok.example": 1,
			},
		},
		{
			name:      "failover to second provider",
			host:      "failover.example",
			table:     &route.Table{Policies: map[string][]string{"failover.example": {"a", "b"}}},
			providers: map[string]string{"a": "a:7070", "b": "b:7070"},
			dial: map[string]func() (net.Conn, error){
				"a:7070": unreachable,
				"b:7070": replying("OK"),
			},
			wantRequests: map[string]float64{
				"network.transport=tcp,tunny.outcome=success,tunny.provider=b,tunny.route=failover.example": 1,
			},
			wantAttempts: map[string]float64{
				"error.type=provider_unreachable,network.transport=tcp,tunny.outcome=error,tunny.provider=a,tunny.route=failover.example": 1,
				"network.transport=tcp,tunny.outcome=success,tunny.provider=b,tunny.route=failover.example":                               1,
			},
		},
		{
			name:      "all providers fail",
			host:      "fail.example",
			table:     &route.Table{Policies: map[string][]string{"fail.example": {"missing", "a", "b"}}},
			providers: map[string]string{"a": "a:7070", "b": "b:7070"},
			dial: map[string]func() (net.Conn, error){
				"a:7070": replying(""),
				"b:7070": replying("ERR dial tcp: connection refused"),
			},
			wantErr: true,
			wantRequests: map[string]float64{
				"error.type=provider_rejected,network.transport=tcp,tunny.outcome=error,tunny.route=fail.example": 1,
			},
			wantAttempts: map[string]float64{
				"error.type=provider_not_found,network.transport=tcp,tunny.outcome=error,tunny.provider=missing,tunny.route=fail.example": 1,
				"error.type=handshake,network.transport=tcp,tunny.outcome=error,tunny.provider=a,tunny.route=fail.example":                1,
				"error.type=provider_rejected,network.transport=tcp,tunny.outcome=error,tunny.provider=b,tunny.route=fail.example":        1,
			},
		},
		{
			name: "IP match is labelled with route hostname",
			host: "203.0.113.10",
			table: &route.Table{
				IPs:   map[string]string{"203.0.113.10": "a"},
				Hosts: map[string]string{"203.0.113.10": "ip.example"},
			},
			providers: map[string]string{"a": "a:7070"},
			dial:      map[string]func() (net.Conn, error){"a:7070": replying("OK")},
			wantRequests: map[string]float64{
				"network.transport=tcp,tunny.outcome=success,tunny.provider=a,tunny.route=ip.example": 1,
			},
			wantAttempts: map[string]float64{
				"network.transport=tcp,tunny.outcome=success,tunny.provider=a,tunny.route=ip.example": 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Dialer{
				Node:      fakeNode{dial: tt.dial},
				Routes:    tt.table,
				Providers: tt.providers,
			}

			conn, err := d.Dial(context.Background(), "tcp", net.JoinHostPort(tt.host, "443"))
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NoError(t, conn.Close())
			}

			route := tt.host
			if host, ok := tt.table.Hosts[tt.host]; ok {
				route = host
			}
			assert.Equal(t, tt.wantRequests, routePoints(t, "tunny.route.request.duration", route))
			assert.Equal(t, tt.wantAttempts, routePoints(t, "tunny.route.provider.attempt.duration", route))
		})
	}
}

func TestDialClassifiesCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	d := &Dialer{
		Node: fakeNode{dial: map[string]func() (net.Conn, error){
			"a:7070": func() (net.Conn, error) { return nil, ctx.Err() },
		}},
		Routes:    &route.Table{Routes: map[string]string{"canceled.example": "a"}},
		Providers: map[string]string{"a": "a:7070"},
	}

	_, err := d.Dial(ctx, "tcp", "canceled.example:443")
	require.Error(t, err)

	assert.Equal(t, map[string]float64{
		"error.type=canceled,network.transport=tcp,tunny.outcome=error,tunny.route=canceled.example": 1,
	}, routePoints(t, "tunny.route.request.duration", "canceled.example"))
}

func TestDialRecordsDirectDial(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			_ = conn.Close()
		}
	}()

	d := &Dialer{Routes: &route.Table{}}
	before := observabilitytest.Points(t, "tunny.direct.dial.duration")["network.transport=tcp,tunny.outcome=success"]

	conn, err := d.Dial(context.Background(), "tcp", listener.Addr().String())
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	after := observabilitytest.Points(t, "tunny.direct.dial.duration")["network.transport=tcp,tunny.outcome=success"]
	assert.Equal(t, before+1, after)
}
