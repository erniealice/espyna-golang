//go:build grpc_vanilla

package interceptors

import (
	"context"

	"google.golang.org/grpc/metadata"
)

const (
	// Context key constants for gRPC context values
	ContextKeyUID        = "uid"
	ContextKeyEmail      = "email"
	ContextKeyIdentity   = "identity"
	ContextKeyWorkspace  = "workspace_id"
	ContextKeyExpires    = "expires"
)

// Metadata keys for gRPC metadata
const (
	MetadataKeyAuthorization    = "authorization"
	MetadataKeyXAPIKey          = "x-api-key"
	MetadataKeyXAPIKeyScheduler = "x-api-key-scheduler"
	MetadataKeyXWorkspaceID     = "x-workspace-id"
)

// ExtractMetadataToContext extracts gRPC metadata and adds it to the context
// This helper is used by the adapter to process incoming metadata before calling handlers
func ExtractMetadataToContext(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}

	// Extract authorization header
	if authValues := md.Get(MetadataKeyAuthorization); len(authValues) > 0 {
		ctx = context.WithValue(ctx, "authorization", authValues[0])
	}

	// Extract X-API-Key
	if apiKeyValues := md.Get(MetadataKeyXAPIKey); len(apiKeyValues) > 0 {
		ctx = context.WithValue(ctx, "x-api-key", apiKeyValues[0])
	}

	// Extract X-API-Key-Scheduler
	if schedulerKeyValues := md.Get(MetadataKeyXAPIKeyScheduler); len(schedulerKeyValues) > 0 {
		ctx = context.WithValue(ctx, "x-api-key-scheduler", schedulerKeyValues[0])
	}

	// Extract X-Workspace-ID
	if workspaceIDValues := md.Get(MetadataKeyXWorkspaceID); len(workspaceIDValues) > 0 {
		ctx = context.WithValue(ctx, ContextKeyWorkspace, workspaceIDValues[0])
	}

	return ctx
}

// SetContextValue sets a value in the context with a typed key
func SetContextValue(ctx context.Context, key string, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}

// GetContextValue retrieves a value from the context
func GetContextValue(ctx context.Context, key string) interface{} {
	return ctx.Value(key)
}
