package customization

import "sync"

// RouteCustomizer manages route path customizations
type RouteCustomizer struct {
	globalPrefix   string
	domainPrefixes map[string]string // domain -> custom prefix
	routePaths     map[string]string // route.Name -> custom path
	mu             sync.RWMutex
}

// CustomizationConfig holds all path overrides (for YAML/JSON loading)
type CustomizationConfig struct {
	GlobalPrefix   string            `json:"globalPrefix" yaml:"globalPrefix"`
	DomainPrefixes map[string]string `json:"domainPrefixes" yaml:"domainPrefixes"`
	RoutePaths     map[string]string `json:"routePaths" yaml:"routePaths"`
}
