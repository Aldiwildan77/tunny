package tunnycontrol

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Aldiwildan77/tunny/observability"
	"github.com/Aldiwildan77/tunny/observability/observabilitytest"
	"github.com/Aldiwildan77/tunny/route"
)

func TestMain(m *testing.M) {
	observabilitytest.Install()
	os.Exit(m.Run())
}

func TestHTTPMetricsCollapseHostnames(t *testing.T) {
	server, err := NewServer(route.New(nil), "proxy", "client", "direct")
	require.NoError(t, err)
	gateway, err := server.gateway()
	require.NoError(t, err)
	handler := httpMetrics(gateway)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/status", nil))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodDelete, "/v1/routes/example.com", nil))

	points := observabilitytest.Points(t, "tunny.control.http.duration")
	assert.Equal(t, float64(1), points["http.request.method=GET,http.response.status_code=200,http.route=/v1/status"])
	assert.Equal(t, float64(1), points["http.request.method=DELETE,http.response.status_code=200,http.route=/v1/routes/{hostname}"])
}

func TestUnaryMetricsRecordsStatusCode(t *testing.T) {
	info := &grpc.UnaryServerInfo{FullMethod: "/tunnycontrol.ControlPlane/SetRoute"}
	_, err := unaryMetrics(context.Background(), nil, info, func(context.Context, any) (any, error) {
		return nil, status.Error(codes.InvalidArgument, "bad")
	})
	require.Error(t, err)

	points := observabilitytest.Points(t, "tunny.control.rpc.duration")
	assert.Equal(t, float64(1), points["rpc.grpc.status_code=InvalidArgument,rpc.method=/tunnycontrol.ControlPlane/SetRoute"])
}

func TestStatusReportsActiveConnections(t *testing.T) {
	server, err := NewServer(route.New(nil), "proxy", "client", "direct")
	require.NoError(t, err)

	client, conn := net.Pipe()
	defer client.Close()
	tracked := observability.TrackConn(conn, observability.IngressSOCKS5, "tcp")

	got, err := server.GetStatus(context.Background(), &GetStatusRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), got.GetActiveConnections())

	require.NoError(t, tracked.Close())
	got, err = server.GetStatus(context.Background(), &GetStatusRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), got.GetActiveConnections())
}
