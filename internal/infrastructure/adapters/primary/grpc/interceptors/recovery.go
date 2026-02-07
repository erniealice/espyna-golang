//go:build grpc_vanilla

package interceptors

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoveryInterceptor provides panic recovery for gRPC requests
type RecoveryInterceptor struct{}

// NewRecoveryInterceptor creates a new recovery interceptor instance
func NewRecoveryInterceptor() *RecoveryInterceptor {
	return &RecoveryInterceptor{}
}

// UnaryInterceptor returns a unary server interceptor that recovers from panics
func (i *RecoveryInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {
		// Recover from panics
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC recovered in gRPC method %s: %v\n%s", info.FullMethod, r, debug.Stack())
			}
		}()

		// Call handler
		resp, err := handler(ctx, req)

		// Check for panic
		if r := recover(); r != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("Internal server error: %v", r))
		}

		return resp, err
	}
}

// recoverHandler is a helper that recovers from panics and returns a gRPC error
func recoverHandler(info *grpc.UnaryServerInfo) (interface{}, error) {
	if r := recover(); r != nil {
		log.Printf("PANIC recovered in gRPC method %s: %v\n%s", info.FullMethod, r, debug.Stack())
		return nil, status.Error(codes.Internal, "Internal server error")
	}
	return nil, nil
}
