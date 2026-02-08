//go:build grpc_vanilla

package vanilla

import (
	"context"
	"fmt"
	"log"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/erniealice/espyna-golang/internal/composition/core"
	"github.com/erniealice/espyna-golang/internal/composition/routing"
	"github.com/erniealice/espyna-golang/internal/composition/routing/customization"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/grpc/interceptors"
)

// EspynaService is a dynamic gRPC service that maps gRPC methods to HTTP routes
type EspynaService struct {
	container     *core.Container
	routes        map[string]*routing.Route // fullMethod -> route
	methodDescs   map[string]*MethodDescriptor
	services      map[string]*ServiceDescriptor // serviceName -> ServiceDescriptor
}

// ServiceDescriptor describes a gRPC service (e.g., ClientService)
type ServiceDescriptor struct {
	Name        string
	FullName    string
	Methods     map[string]*MethodDescriptor
}

// MethodDescriptor describes a gRPC method
type MethodDescriptor struct {
	Name           string
	FullMethod     string // /espyna.entity.v1.ClientService/Create
	InputType      string
	OutputType     string
	HTTPRoute      *routing.Route
}

// NewEspynaService creates a new Espyna dynamic service
func NewEspynaService(container *core.Container) *EspynaService {
	service := &EspynaService{
		container:   container,
		routes:      make(map[string]*routing.Route),
		methodDescs: make(map[string]*MethodDescriptor),
		services:    make(map[string]*ServiceDescriptor),
	}
	service.buildRouteMap()
	return service
}

// buildRouteMap converts all HTTP routes to gRPC method names
func (s *EspynaService) buildRouteMap() {
	customizer := customization.NewRouteCustomizer()
	baseRoutes := s.container.GetRouteManager().GetAllRoutes()
	routes := customizer.ApplyCustomizations(baseRoutes)

	log.Printf("Building gRPC route map from %d HTTP routes", len(routes))

	for _, route := range routes {
		grpcMethod := s.httpRouteToGRPCMethod(route)
		if grpcMethod != "" {
			s.routes[grpcMethod] = route

			// Create method descriptor
			desc := &MethodDescriptor{
				FullMethod: grpcMethod,
				HTTPRoute:  route,
			}

			// Parse method name and service name
			parts := strings.Split(strings.Trim(grpcMethod, "/"), "/")
			if len(parts) == 2 {
				desc.Name = parts[1]
				serviceFullName := parts[0]

				// Create service descriptor if not exists
				if _, exists := s.services[serviceFullName]; !exists {
					s.services[serviceFullName] = &ServiceDescriptor{
						Name:     s.extractServiceName(serviceFullName),
						FullName: serviceFullName,
						Methods:  make(map[string]*MethodDescriptor),
					}
				}

				// Add method to service
				s.services[serviceFullName].Methods[desc.Name] = desc
			}

			s.methodDescs[grpcMethod] = desc
			log.Printf("Mapped gRPC method: %s -> %s %s", grpcMethod, route.Method, route.Path)
		}
	}

	log.Printf("Total gRPC services: %d, methods: %d", len(s.services), len(s.routes))
}

// httpRouteToGRPCMethod converts route metadata to gRPC full method name
//
// Mapping Formula:
//   Domain: route.Metadata.Domain (e.g., "entity")
//   Resource: route.Metadata.Resource (e.g., "client")
//   Operation: route.Metadata.Operation (e.g., "create")
//
//   Service: espyna.{domain}.v1.{Resource}Service
//   Method: {Operation}
//
// Examples:
//   /api/entity/client/create -> /espyna.entity.v1.ClientService/Create
//   /api/entity/client/list   -> /espyna.entity.v1.ClientService/List
//   /api/subscription/plan/read -> /espyna.subscription.v1.PlanService/Read
func (s *EspynaService) httpRouteToGRPCMethod(route *routing.Route) string {
	if route.Metadata.Domain == "" || route.Metadata.Resource == "" || route.Metadata.Operation == "" {
		return ""
	}

	domain := route.Metadata.Domain
	resource := capitalize(route.Metadata.Resource)
	operation := route.Metadata.Operation

	// Build gRPC method: /espyna.{domain}.v1.{Resource}Service/{Operation}
	// Note: We use {Resource}Service, not just {Resource}
	return fmt.Sprintf("/espyna.%s.v1.%sService/%s", domain, resource, operation)
}

// extractServiceName extracts the short service name from full name
// e.g., "espyna.entity.v1.ClientService" -> "ClientService"
func (s *EspynaService) extractServiceName(fullName string) string {
	parts := strings.Split(fullName, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullName
}

// Register registers the Espyna service with the gRPC server
// Since gRPC requires method descriptors at registration time, we use
// the grpc.ServiceDesc approach with dynamic method registration
func (s *EspynaService) Register(server *grpc.Server) {
	// For each service, create and register a service descriptor
	for serviceName, serviceDesc := range s.services {
		s.registerService(server, serviceName, serviceDesc)
	}

	log.Printf("EspynaService registered %d services", len(s.services))
}

// registerService registers a single service with the gRPC server
func (s *EspynaService) registerService(server *grpc.Server, fullName string, desc *ServiceDescriptor) {
	// Build method handlers
	methods := make([]grpc.MethodDesc, 0, len(desc.Methods))
	streams := make([]grpc.StreamDesc, 0)

	for methodName, methodDesc := range desc.Methods {
		method := grpc.MethodDesc{
			Name: methodName,
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				// Extract the method name from context (set by interceptor)
				// In a real implementation, you'd pass this through closure context
				fullMethod := fullName + "/" + methodName

				// Decode the request
				var req proto.Message
				if err := dec(&req); err != nil {
					return nil, err
				}

				// Execute the method
				return s.executeMethod(ctx, fullMethod, req)
			},
		}
		methods = append(methods, method)
	}

	// Create service description
	serviceDesc := grpc.ServiceDesc{
		ServiceName: fullName,
		Methods:     methods,
		Streams:     streams,
		Metadata:    "espyna.proto",
	}

	// Register the service
	server.RegisterService(&serviceDesc, s)
}

// executeMethod executes a gRPC method by name
func (s *EspynaService) executeMethod(ctx context.Context, fullMethod string, req interface{}) (proto.Message, error) {
	route, ok := s.routes[fullMethod]
	if !ok {
		return nil, status.Errorf(codes.Unimplemented, "method not found: %s", fullMethod)
	}

	// Extract metadata to context
	ctx = interceptors.ExtractMetadataToContext(ctx)

	// Add default workspace context for testing
	if ctx.Value("workspace_id") == nil {
		ctx = context.WithValue(ctx, "workspace_id", "test-workspace")
	}

	// Convert request to proto.Message if needed
	var protoReq proto.Message
	if pm, ok := req.(proto.Message); ok {
		protoReq = pm
	}

	// Execute handler
	resp, err := route.Handler.Execute(ctx, protoReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "handler execution failed: %v", err)
	}

	// Convert response to proto.Message
	if pm, ok := resp.(proto.Message); ok {
		return pm, nil
	}

	return nil, status.Errorf(codes.Internal, "invalid response type")
}

// GetRoutes returns the registered routes
func (s *EspynaService) GetRoutes() map[string]*routing.Route {
	return s.routes
}

// GetServices returns the registered services
func (s *EspynaService) GetServices() map[string]*ServiceDescriptor {
	return s.services
}

// GetMethodDescriptor returns a method descriptor by full method name
func (s *EspynaService) GetMethodDescriptor(fullMethod string) (*MethodDescriptor, bool) {
	desc, ok := s.methodDescs[fullMethod]
	return desc, ok
}

// capitalize capitalizes the first letter of a string
func capitalize(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

// toLowerCamel converts a string to lowerCamelCase
func toLowerCamel(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(string(s[0])) + s[1:]
}
