package tunnycontrol

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Aldiwildan77/tunny/route"
)

func TestServerRoutes(t *testing.T) {
	server, err := NewServer(
		route.New(map[string]string{"example.com": "provider-1"}),
		"proxy",
		"client",
		"direct",
	)
	if err != nil {
		t.Fatal(err)
	}

	routes, err := server.ListRoutes(context.Background(), &ListRoutesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(routes.GetRoutes()) != 1 {
		t.Fatalf("ListRoutes() returned %d routes, want 1", len(routes.GetRoutes()))
	}

	created, err := server.SetRoute(context.Background(), &SetRouteRequest{
		Hostname: "new.example.com",
		Provider: "provider-2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.GetHostname() != "new.example.com" {
		t.Fatalf("SetRoute() hostname = %q, want %q", created.GetHostname(), "new.example.com")
	}

	if _, err := server.DeleteRoute(context.Background(), &DeleteRouteRequest{
		Hostname: "new.example.com",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestServerStatus(t *testing.T) {
	server, err := NewServer(route.New(nil), "tunnel", "client", "direct")
	if err != nil {
		t.Fatal(err)
	}

	status, err := server.GetStatus(context.Background(), &GetStatusRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if status.GetMode() != "tunnel" || status.GetNodeName() != "client" {
		t.Fatalf("GetStatus() = mode %q node %q", status.GetMode(), status.GetNodeName())
	}
}

func TestGatewayStatusAndSetRoute(t *testing.T) {
	server, err := NewServer(route.New(nil), "proxy", "client", "direct")
	if err != nil {
		t.Fatal(err)
	}

	gateway, err := server.gateway()
	if err != nil {
		t.Fatal(err)
	}

	statusRequest := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	statusResponse := httptest.NewRecorder()
	gateway.ServeHTTP(statusResponse, statusRequest)
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("GET /v1/status status = %d, want %d", statusResponse.Code, http.StatusOK)
	}
	if !strings.Contains(statusResponse.Body.String(), `"node_name":"client"`) {
		t.Fatalf("GET /v1/status body = %s", statusResponse.Body.String())
	}

	setRequest := httptest.NewRequest(
		http.MethodPost,
		"/v1/routes",
		strings.NewReader(`{"hostname":"example.com","provider":"provider-1"}`),
	)
	setResponse := httptest.NewRecorder()
	gateway.ServeHTTP(setResponse, setRequest)
	if setResponse.Code != http.StatusOK {
		t.Fatalf("POST /v1/routes status = %d, want %d", setResponse.Code, http.StatusOK)
	}
}
