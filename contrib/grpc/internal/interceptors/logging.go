//go:build grpc_vanilla

package interceptors

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

// LoggingInterceptor provides request/response logging for gRPC requests
type LoggingInterceptor struct{}

// NewLoggingInterceptor creates a new logging interceptor instance
func NewLoggingInterceptor() *LoggingInterceptor {
	return &LoggingInterceptor{}
}

// UnaryInterceptor returns a unary server interceptor that logs requests and responses
func (i *LoggingInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Log request
		log.Printf("GRPC_REQUEST: method=%s", info.FullMethod)

		// Call handler
		resp, err := handler(ctx, req)

		// Log response
		duration := time.Since(start)
		if err != nil {
			log.Printf("GRPC_RESPONSE: method=%s status=error duration=%s error=%v", info.FullMethod, duration, err)
		} else {
			log.Printf("GRPC_RESPONSE: method=%s status=ok duration=%s", info.FullMethod, duration)
		}

		return resp, err
	}
}
