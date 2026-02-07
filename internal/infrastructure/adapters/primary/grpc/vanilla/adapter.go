//go:build grpc_vanilla

package vanilla

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/composition/core"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/primary/grpc/interceptors"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterServerProvider(
		"grpc_vanilla",
		func() ports.ServerProvider {
			return NewGRPCVanillaAdapter()
		},
		buildFromEnv,
	)
}

// buildFromEnv creates a gRPC Vanilla adapter from environment variables.
func buildFromEnv() (ports.ServerProvider, error) {
	adapter := NewGRPCVanillaAdapter()
	return adapter, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// GRPCVanillaAdapter implements ServerProvider for gRPC server.
type GRPCVanillaAdapter struct {
	server    *grpc.Server
	container *core.Container
	enabled   bool
	listener  net.Listener

	// Interceptors
	authInterceptor     *interceptors.AuthenticationInterceptor
	recoveryInterceptor *interceptors.RecoveryInterceptor
	loggingInterceptor  *interceptors.LoggingInterceptor
}

// NewGRPCVanillaAdapter creates a new gRPC server adapter.
func NewGRPCVanillaAdapter() *GRPCVanillaAdapter {
	return &GRPCVanillaAdapter{}
}

// Name returns the provider name.
func (a *GRPCVanillaAdapter) Name() string {
	return "grpc_vanilla"
}

// Initialize sets up the gRPC server with the container.
// The container parameter should be *core.Container but is typed as any
// to satisfy the ports.ServerProvider interface and avoid import cycles.
func (a *GRPCVanillaAdapter) Initialize(container any) error {
	if container == nil {
		return fmt.Errorf("grpc vanilla adapter requires a non-nil container")
	}

	// Type assert to *core.Container
	c, ok := container.(*core.Container)
	if !ok {
		return fmt.Errorf("grpc vanilla adapter requires *core.Container, got %T", container)
	}

	a.container = c
	a.enabled = true

	// Create interceptors
	a.recoveryInterceptor = interceptors.NewRecoveryInterceptor()
	a.loggingInterceptor = interceptors.NewLoggingInterceptor()

	// Get auth service from provider manager
	authService := ports.AuthService(nil)
	if provider := c.GetAuthProvider(); provider != nil {
		if auth, ok := provider.(ports.AuthService); ok {
			authService = auth
		}
	}
	a.authInterceptor = interceptors.NewAuthenticationInterceptor(authService)

	// Create gRPC server with interceptor chain
	// Order: Recovery -> Logging -> Authentication
	a.server = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			a.recoveryInterceptor.UnaryInterceptor(),
			a.loggingInterceptor.UnaryInterceptor(),
			a.authInterceptor.UnaryInterceptor(),
		),
	)

	// Register health service
	healthServer := NewHealthServer()
	grpc_health_v1.RegisterHealthServer(a.server, healthServer)

	// Register Espyna dynamic service
	espynaService := NewEspynaService(c)
	espynaService.Register(a.server)

	// Enable gRPC reflection if configured
	if getEnv("GRPC_REFLECTION_ENABLED", "true") == "true" {
		reflection.Register(a.server)
		log.Printf("gRPC reflection enabled")
	}

	log.Printf("gRPC Vanilla adapter initialized successfully")
	return nil
}

// Start starts the gRPC server on the specified address.
func (a *GRPCVanillaAdapter) Start(addr string) error {
	if a.server == nil {
		return fmt.Errorf("grpc vanilla adapter not initialized - call Initialize() first")
	}

	printServerInfo("grpc_vanilla", addr)

	// Create listener
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	a.listener = listener

	log.Printf("gRPC server starting on %s", addr)
	return a.server.Serve(listener)
}

// IsHealthy checks if the server is healthy.
func (a *GRPCVanillaAdapter) IsHealthy(ctx context.Context) error {
	if a.server == nil {
		return fmt.Errorf("grpc server not initialized")
	}
	return nil
}

// Close shuts down the gRPC server.
func (a *GRPCVanillaAdapter) Close() error {
	if a.server != nil {
		log.Printf("gRPC Vanilla adapter closing")
		a.server.GracefulStop()
	}
	if a.listener != nil {
		return a.listener.Close()
	}
	return nil
}

// IsEnabled returns whether this adapter is enabled.
func (a *GRPCVanillaAdapter) IsEnabled() bool {
	return a.enabled
}

// GetServer returns the underlying gRPC server for advanced customization.
func (a *GRPCVanillaAdapter) GetServer() *grpc.Server {
	return a.server
}

// Compile-time interface check
var _ ports.ServerProvider = (*GRPCVanillaAdapter)(nil)

// =============================================================================
// Health Server Implementation
// =============================================================================

// HealthServer implements the gRPC health checking service
type HealthServer struct {
	grpc_health_v1.UnimplementedHealthServer
}

// NewHealthServer creates a new health server
func NewHealthServer() *HealthServer {
	return &HealthServer{}
}

// Check implements the health check RPC
func (s *HealthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

// Watch implements the health watch RPC
func (s *HealthServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	// Send a single serving status and close
	return stream.Send(&grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	})
}
