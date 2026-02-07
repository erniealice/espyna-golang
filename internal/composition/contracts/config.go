package contracts

import (
	"time"
)

// ============================================================================
// Main Configuration Types
// ============================================================================

// Config represents the overall routing configuration
type Config struct {
	// Global configuration
	Version     string          `json:"version" yaml:"version"`
	Title       string          `json:"title" yaml:"title"`
	Description string          `json:"description" yaml:"description"`
	BasePath    string          `json:"basePath" yaml:"basePath"`
	CORS        CORSConfig      `json:"cors" yaml:"cors"`
	RateLimit   RateLimitConfig `json:"rateLimit" yaml:"rateLimit"`
	Timeout     time.Duration   `json:"timeout" yaml:"timeout"`

	// Feature flags
	EnableMetrics     bool `json:"enableMetrics" yaml:"enableMetrics"`
	EnableHealthCheck bool `json:"enableHealthCheck" yaml:"enableHealthCheck"`
	EnableAuth        bool `json:"enableAuth" yaml:"enableAuth"`
	EnableAuditLog    bool `json:"enableAuditLog" yaml:"enableAuditLog"`

	// Domain configurations
	Domains map[string]DomainConfig `json:"domains" yaml:"domains"`

	// Migration settings
	MigrationConfig MigrationConfig `json:"migration" yaml:"migration"`
}

// ============================================================================
// CORS Configuration
// ============================================================================

// CORSConfig represents CORS configuration
type CORSConfig struct {
	Enabled          bool     `json:"enabled" yaml:"enabled"`
	AllowedOrigins   []string `json:"allowedOrigins" yaml:"allowedOrigins"`
	AllowedMethods   []string `json:"allowedMethods" yaml:"allowedMethods"`
	AllowedHeaders   []string `json:"allowedHeaders" yaml:"allowedHeaders"`
	ExposedHeaders   []string `json:"exposedHeaders" yaml:"exposedHeaders"`
	AllowCredentials bool     `json:"allowCredentials" yaml:"allowCredentials"`
	MaxAge           int      `json:"maxAge" yaml:"maxAge"`
}

// ============================================================================
// Rate Limiting Configuration
// ============================================================================

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Global rate limit
	RequestsPerMinute int `json:"requestsPerMinute" yaml:"requestsPerMinute"`
	BurstSize         int `json:"burstSize" yaml:"burstSize"`

	// Per-endpoint rate limits
	Endpoints map[string]EndpointRateLimit `json:"endpoints" yaml:"endpoints"`
}

// EndpointRateLimit represents rate limit for specific endpoints
type EndpointRateLimit struct {
	RequestsPerMinute int `json:"requestsPerMinute" yaml:"requestsPerMinute"`
	BurstSize         int `json:"burstSize" yaml:"burstSize"`
}

// ============================================================================
// Domain Configuration
// ============================================================================

// DomainConfig represents configuration for a specific domain
type DomainConfig struct {
	Enabled    bool                      `json:"enabled" yaml:"enabled"`
	Prefix     string                    `json:"prefix" yaml:"prefix"`
	Middleware []string                  `json:"middleware" yaml:"middleware"`
	Endpoints  map[string]EndpointConfig `json:"endpoints" yaml:"endpoints"`
}

// EndpointConfig represents configuration for a specific endpoint
type EndpointConfig struct {
	Enabled    bool             `json:"enabled" yaml:"enabled"`
	Methods    []string         `json:"methods" yaml:"methods"`
	Middleware []string         `json:"middleware" yaml:"middleware"`
	Auth       AuthConfig       `json:"auth" yaml:"auth"`
	Cache      CacheConfig      `json:"cache" yaml:"cache"`
	Validation ValidationConfig `json:"validation" yaml:"validation"`
}

// AuthConfig represents authentication configuration for an endpoint
type AuthConfig struct {
	Required bool     `json:"required" yaml:"required"`
	Roles    []string `json:"roles" yaml:"roles"`
	Scopes   []string `json:"scopes" yaml:"scopes"`
}

// CacheConfig represents caching configuration for an endpoint
type CacheConfig struct {
	Enabled bool          `json:"enabled" yaml:"enabled"`
	TTL     time.Duration `json:"ttl" yaml:"ttl"`
	Key     string        `json:"key" yaml:"key"`
}

// ValidationConfig represents validation configuration for an endpoint
type ValidationConfig struct {
	Enabled     bool     `json:"enabled" yaml:"enabled"`
	StrictMode  bool     `json:"strictMode" yaml:"strictMode"`
	CustomRules []string `json:"customRules" yaml:"customRules"`
}

// ============================================================================
// Migration Configuration
// ============================================================================

// MigrationConfig represents configuration for gradual migration
type MigrationConfig struct {
	// Enable legacy route compatibility
	EnableLegacyRoutes bool `json:"enableLegacyRoutes" yaml:"enableLegacyRoutes"`

	// Migration strategy: "parallel", "gradual", "blue-green"
	Strategy string `json:"strategy" yaml:"strategy"`

	// Route mapping from old to new
	RouteMappings map[string]string `json:"routeMappings" yaml:"routeMappings"`

	// Traffic splitting for gradual migration
	TrafficSplitting TrafficSplitConfig `json:"trafficSplitting" yaml:"trafficSplitting"`
}

// TrafficSplitConfig represents traffic splitting configuration
type TrafficSplitConfig struct {
	Enabled          bool        `json:"enabled" yaml:"enabled"`
	NewSystemPercent int         `json:"newSystemPercent" yaml:"newSystemPercent"`
	Rules            []SplitRule `json:"rules" yaml:"rules"`
}

// SplitRule represents a rule for traffic splitting
type SplitRule struct {
	Condition string `json:"condition" yaml:"condition"` // e.g., "header:X-Test-Group=beta"
	Percent   int    `json:"percent" yaml:"percent"`     // Percentage to route to new system
}

// ============================================================================
// Config Methods
// ============================================================================

// Validate validates the routing configuration and sets defaults
func (c *Config) Validate() error {
	if c.BasePath == "" {
		c.BasePath = "/api"
	}

	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}

	// Validate domain configurations
	for domainName, domainConfig := range c.Domains {
		if domainConfig.Prefix == "" {
			domainConfig.Prefix = "/" + domainName
			c.Domains[domainName] = domainConfig
		}
	}

	return nil
}

// GetEnabledDomains returns a list of enabled domains
func (c *Config) GetEnabledDomains() []string {
	var enabled []string
	for name, config := range c.Domains {
		if config.Enabled {
			enabled = append(enabled, name)
		}
	}
	return enabled
}

// IsEndpointEnabled checks if a specific endpoint is enabled
func (c *Config) IsEndpointEnabled(domain, endpoint string) bool {
	domainConfig, exists := c.Domains[domain]
	if !exists || !domainConfig.Enabled {
		return false
	}

	endpointConfig, exists := domainConfig.Endpoints[endpoint]
	if !exists {
		return true // Default to enabled if not specified
	}

	return endpointConfig.Enabled
}
