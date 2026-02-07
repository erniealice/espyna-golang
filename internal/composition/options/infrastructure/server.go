package infrastructure

import (
	"fmt"
	"strings"
)

// =============================================================================
// SERVER CONFIGURATION TYPES
// =============================================================================

// ServerConfig is a union type that can hold any server configuration
type ServerConfig struct {
	Gin     *GinServerConfig
	Fiber   *FiberServerConfig
	Vanilla *VanillaServerConfig
}

// GinServerConfig defines configuration for Gin web framework
type GinServerConfig struct {
	Host          string `json:"host"`
	Port          string `json:"port"`
	UseNewRouting bool   `json:"use_new_routing"`
}

// Validate validates the gin server configuration
func (c GinServerConfig) Validate() error {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == "" {
		c.Port = "8080"
	}
	return nil
}

// FiberServerConfig defines configuration for Fiber web framework
type FiberServerConfig struct {
	Host          string `json:"host"`
	Port          string `json:"port"`
	UseNewRouting bool   `json:"use_new_routing"`
}

// Validate validates the fiber server configuration
func (c FiberServerConfig) Validate() error {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == "" {
		c.Port = "8080"
	}
	return nil
}

// VanillaServerConfig defines configuration for vanilla net/http server
type VanillaServerConfig struct {
	Host          string `json:"host"`
	Port          string `json:"port"`
	UseNewRouting bool   `json:"use_new_routing"`
}

// Validate validates the vanilla server configuration
func (c VanillaServerConfig) Validate() error {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == "" {
		c.Port = "8080"
	}
	return nil
}

// =============================================================================
// ENVIRONMENT CONFIGURATION LOADERS
// =============================================================================

func createGinConfigFromEnv() GinServerConfig {
	return GinServerConfig{
		Host:          GetEnv("GIN_HOST", GetEnv("SERVER_HOST", "localhost")),
		Port:          GetEnv("GIN_PORT", GetEnv("SERVER_PORT", "8080")),
		UseNewRouting: GetEnv("GIN_USE_NEW_ROUTING", "false") == "true",
	}
}

func createFiberConfigFromEnv() FiberServerConfig {
	return FiberServerConfig{
		Host:          GetEnv("FIBER_HOST", GetEnv("SERVER_HOST", "localhost")),
		Port:          GetEnv("FIBER_PORT", GetEnv("SERVER_PORT", "8080")),
		UseNewRouting: GetEnv("FIBER_USE_NEW_ROUTING", "false") == "true",
	}
}

func createVanillaConfigFromEnv() VanillaServerConfig {
	return VanillaServerConfig{
		Host:          GetEnv("VANILLA_HOST", GetEnv("SERVER_HOST", "localhost")),
		Port:          GetEnv("VANILLA_PORT", GetEnv("SERVER_PORT", "8080")),
		UseNewRouting: GetEnv("VANILLA_USE_NEW_ROUTING", "false") == "true",
	}
}

// =============================================================================
// SERVER FRAMEWORK OPTIONS
// =============================================================================

// WithServerFromEnv dynamically selects server framework based on CONFIG_SERVER_FRAMEWORK
func WithServerFromEnv() ContainerOption {
	return func(c Container) error {
		framework := strings.ToLower(GetEnv("CONFIG_SERVER_FRAMEWORK", "vanilla"))

		switch framework {
		case "gin":
			return WithGinServer(createGinConfigFromEnv())(c)
		case "fiber":
			return WithFiberServer(createFiberConfigFromEnv())(c)
		case "vanilla", "":
			return WithVanillaServer(createVanillaConfigFromEnv())(c)
		default:
			return fmt.Errorf("unsupported server framework: %s", framework)
		}
	}
}

// WithGinServer configures Gin web framework
func WithGinServer(config GinServerConfig) ContainerOption {
	return func(c Container) error {
		if setter, ok := c.(ServerConfigSetter); ok {
			setter.SetServerConfig(ServerConfig{Gin: &config})
		}
		fmt.Printf("🚀 Configured Gin server: %s:%s (new routing: %t)\n", config.Host, config.Port, config.UseNewRouting)
		return nil
	}
}

// WithFiberServer configures Fiber web framework
func WithFiberServer(config FiberServerConfig) ContainerOption {
	return func(c Container) error {
		if setter, ok := c.(ServerConfigSetter); ok {
			setter.SetServerConfig(ServerConfig{Fiber: &config})
		}
		fmt.Printf("🚀 Configured Fiber server: %s:%s (new routing: %t)\n", config.Host, config.Port, config.UseNewRouting)
		return nil
	}
}

// WithVanillaServer configures vanilla net/http server
func WithVanillaServer(config VanillaServerConfig) ContainerOption {
	return func(c Container) error {
		if setter, ok := c.(ServerConfigSetter); ok {
			setter.SetServerConfig(ServerConfig{Vanilla: &config})
		}
		fmt.Printf("🚀 Configured vanilla server: %s:%s (new routing: %t)\n", config.Host, config.Port, config.UseNewRouting)
		return nil
	}
}
