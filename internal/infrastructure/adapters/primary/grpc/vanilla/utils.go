//go:build grpc_vanilla

package vanilla

import (
	"fmt"
	"os"
	"strings"
)

// getEnv returns environment variable value or default if not set
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// printServerInfo prints server startup information
func printServerInfo(framework, addr string) {
	fmt.Printf("\n")
	fmt.Printf("  Espyna Server\n")
	fmt.Printf("  Framework: %s\n", framework)
	fmt.Printf("  Address: %s\n", addr)
	fmt.Printf("  Database: %s\n", getEnv("CONFIG_DATABASE_PROVIDER", "mock_db"))
	fmt.Printf("  Auth: %s\n", getEnv("CONFIG_AUTH_PROVIDER", "mock_auth"))
	fmt.Printf("  ID: %s\n", getEnv("CONFIG_ID_PROVIDER", "noop"))
	fmt.Printf("  Storage: %s\n", getEnv("CONFIG_STORAGE_PROVIDER", "mock_storage"))
	fmt.Printf("\n")
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

// toSnakeCase converts a string to snake_case
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

// getGRPCEnv returns gRPC-specific environment variable value or default
func getGRPCEnv(key, defaultValue string) string {
	envKey := "GRPC_" + key
	return getEnv(envKey, defaultValue)
}

// isReflectionEnabled returns true if gRPC reflection is enabled
func isReflectionEnabled() bool {
	return getGRPCEnv("REFLECTION_ENABLED", "true") == "true"
}

// getDefaultPort returns the default gRPC port
func getDefaultPort() string {
	return getEnv("SERVER_PORT", "50051")
}

// normalizeMethodName normalizes a gRPC method name
func normalizeMethodName(method string) string {
	// Remove leading slash if present
	method = strings.TrimPrefix(method, "/")
	// Ensure consistent format
	return "/" + method
}

// parseServiceName extracts service name from full method
// e.g., "/espyna.entity.v1.ClientService/Create" -> "espyna.entity.v1.ClientService"
func parseServiceName(fullMethod string) string {
	parts := strings.Split(strings.Trim(fullMethod, "/"), "/")
	if len(parts) > 0 {
		return "/" + parts[0]
	}
	return ""
}

// parseMethodName extracts method name from full method
// e.g., "/espyna.entity.v1.ClientService/Create" -> "Create"
func parseMethodName(fullMethod string) string {
	parts := strings.Split(strings.Trim(fullMethod, "/"), "/")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// domainFromServiceName extracts domain from service name
// e.g., "espyna.entity.v1.ClientService" -> "entity"
func domainFromServiceName(serviceName string) string {
	parts := strings.Split(strings.Trim(serviceName, "/"), ".")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// resourceFromServiceName extracts resource name from service name
// e.g., "espyna.entity.v1.ClientService" -> "ClientService"
func resourceFromServiceName(serviceName string) string {
	parts := strings.Split(strings.Trim(serviceName, "/"), ".")
	if len(parts) >= 4 {
		return parts[3]
	}
	return ""
}
