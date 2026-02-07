package infrastructure

import (
	"fmt"
	"os"
)

// =============================================================================
// CONTAINER INTERFACE (to avoid import cycles)
// =============================================================================

// Container defines the interface for container configuration.
// This avoids import cycles between packages.
type Container interface {
	GetConfig() interface{}
	SetConfig(interface{})
}

// ContainerOption defines a function that can configure a Container.
// This is the core of the functional options pattern.
type ContainerOption func(Container) error

// =============================================================================
// CONFIG SETTER INTERFACES
// =============================================================================

// DatabaseConfigSetter defines methods for setting database configuration
type DatabaseConfigSetter interface {
	SetDatabaseConfig(config interface{})
}

// DatabaseTableConfigSetter defines methods for setting database table configuration
type DatabaseTableConfigSetter interface {
	SetDatabaseTableConfig(config interface{})
}

// AuthConfigSetter defines methods for setting auth configuration
type AuthConfigSetter interface {
	SetAuthConfig(config interface{})
}

// StorageConfigSetter defines methods for setting storage configuration
type StorageConfigSetter interface {
	SetStorageConfig(config interface{})
}

// IDConfigSetter defines methods for setting ID provider configuration
type IDConfigSetter interface {
	SetIDConfig(config interface{})
}

// ServerConfigSetter defines methods for setting server configuration
type ServerConfigSetter interface {
	SetServerConfig(config interface{})
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

// GetEnv retrieves environment variable with default value
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ParseInt converts string to int with default value
func ParseInt(s string) int {
	if s == "" {
		return 0
	}
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}
