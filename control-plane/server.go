package tunnycontrol

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Aldiwildan77/tunny/route"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	UnimplementedControlPlaneServer

	routes            *route.Table
	mode              string
	nodeName          string
	transport         string
	startedAt         time.Time
	activeConnections atomic.Int64
}

func NewServer(routes *route.Table, mode, nodeName, transport string) (*Server, error) {
	if routes == nil {
		return nil, fmt.Errorf("routes cannot be nil")
	}

	return &Server{
		routes:    routes,
		mode:      mode,
		nodeName:  nodeName,
		transport: transport,
		startedAt: time.Now(),
	}, nil
}

func (s *Server) Serve(ctx context.Context, grpcAddress, httpAddress string) error {
	grpcListener, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		return err
	}

	httpListener, err := net.Listen("tcp", httpAddress)
	if err != nil {
		_ = grpcListener.Close()
		return err
	}

	grpcServer := grpc.NewServer()
	RegisterControlPlaneServer(grpcServer, s)

	gateway, err := s.gateway()
	if err != nil {
		_ = grpcListener.Close()
		_ = httpListener.Close()
		return err
	}

	httpServer := &http.Server{Handler: gateway}
	serveErr := make(chan error, 2)

	go func() {
		serveErr <- grpcServer.Serve(grpcListener)
	}()
	go func() {
		serveErr <- httpServer.Serve(httpListener)
	}()

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
		_ = httpServer.Shutdown(context.Background())
	}()

	select {
	case err := <-serveErr:
		if err == http.ErrServerClosed {
			return nil
		}
		grpcServer.GracefulStop()
		_ = httpServer.Shutdown(context.Background())
		return err
	case <-ctx.Done():
		return nil
	}
}

func (s *Server) gateway() (http.Handler, error) {
	mux := runtime.NewServeMux()
	if err := RegisterControlPlaneHandlerServer(context.Background(), mux, s); err != nil {
		return nil, err
	}

	return mux, nil
}

func (s *Server) GetStatus(context.Context, *GetStatusRequest) (*Status, error) {
	return &Status{
		Mode:              s.mode,
		NodeName:          s.nodeName,
		Transport:         s.transport,
		UptimeSeconds:     int64(time.Since(s.startedAt).Seconds()),
		ActiveConnections: s.activeConnections.Load(),
	}, nil
}

func (s *Server) ListRoutes(context.Context, *ListRoutesRequest) (*RouteList, error) {
	snapshot := s.routes.Snapshot()
	hostnames := make([]string, 0, len(snapshot))
	for hostname := range snapshot {
		hostnames = append(hostnames, hostname)
	}
	sort.Strings(hostnames)

	routes := make([]*Route, 0, len(hostnames))
	for _, hostname := range hostnames {
		routes = append(routes, &Route{
			Hostname: hostname,
			Provider: snapshot[hostname],
		})
	}

	return &RouteList{Routes: routes}, nil
}

func (s *Server) SetRoute(_ context.Context, request *SetRouteRequest) (*Route, error) {
	hostname := strings.TrimSpace(strings.ToLower(request.GetHostname()))
	provider := strings.TrimSpace(request.GetProvider())
	if hostname == "" || provider == "" {
		return nil, status.Error(codes.InvalidArgument, "hostname and provider are required")
	}

	s.routes.AddRoute(hostname, provider)
	return &Route{Hostname: hostname, Provider: provider}, nil
}

func (s *Server) DeleteRoute(_ context.Context, request *DeleteRouteRequest) (*DeleteRouteResponse, error) {
	hostname := strings.TrimSpace(strings.ToLower(request.GetHostname()))
	if hostname == "" {
		return nil, status.Error(codes.InvalidArgument, "hostname is required")
	}

	s.routes.RemoveRoute(hostname)
	return &DeleteRouteResponse{}, nil
}

var _ ControlPlaneServer = (*Server)(nil)
