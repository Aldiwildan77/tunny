package gateway

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Aldiwildan77/tunny/observability/observabilitytest"
)

func TestMain(m *testing.M) {
	observabilitytest.Install()
	os.Exit(m.Run())
}

type failingDialer struct{}

func (failingDialer) Dial(context.Context, string, string) (net.Conn, error) {
	return nil, errors.New("unreachable")
}

func TestHTTPProxyRecordsRequestMetrics(t *testing.T) {
	proxy := &HTTPProxy{Dialer: failingDialer{}}
	before := observabilitytest.Points(t, "tunny.http.request.duration")

	badGateway := httptest.NewRecorder()
	proxy.handle(badGateway, httptest.NewRequest(http.MethodGet, "http://example.com/", nil))
	assert.Equal(t, http.StatusBadGateway, badGateway.Code)

	badRequest := httptest.NewRecorder()
	request := httptest.NewRequest("BREW", "/", nil)
	request.Host = ""
	proxy.handle(badRequest, request)
	assert.Equal(t, http.StatusBadRequest, badRequest.Code)

	after := observabilitytest.Points(t, "tunny.http.request.duration")
	assert.Equal(t, before["http.request.method=GET,http.response.status_code=502"]+1, after["http.request.method=GET,http.response.status_code=502"])
	assert.Equal(t, before["http.request.method=_OTHER,http.response.status_code=400"]+1, after["http.request.method=_OTHER,http.response.status_code=400"])
}
