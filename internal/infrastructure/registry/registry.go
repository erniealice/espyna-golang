package registry

import (
	"fmt"
	"sync"
)

// =============================================================================
// Generic Factory Registry
// =============================================================================
//
// FactoryRegistry provides a generic, type-safe registry for provider factories.
// This eliminates code duplication across the 5 provider types (Database, Auth,
// Storage, Email, Payment) by using Go generics (Go 1.18+).
//
// Each provider type gets its own typed registry instance, but all share
// the same underlying implementation.
//
// =============================================================================

// FactoryRegistry is a generic registry for provider factories and builders.
// T is the provider interface type (e.g., ports.DatabaseProvider).
// C is the config proto type (e.g., *dbpb.DatabaseProviderConfig).
type FactoryRegistry[T any, C any] struct {
	factories        map[string]func() T
	configTransforms map[string]func(map[string]any) (C, error)
	buildFromEnv     map[string]func() (T, error)
	mutex            sync.RWMutex
	providerType     string // for error messages
}

// NewFactoryRegistry creates a new generic factory registry.
func NewFactoryRegistry[T any, C any](providerType string) *FactoryRegistry[T, C] {
	return &FactoryRegistry[T, C]{
		factories:        make(map[string]func() T),
		configTransforms: make(map[string]func(map[string]any) (C, error)),
		buildFromEnv:     make(map[string]func() (T, error)),
		providerType:     providerType,
	}
}

// RegisterFactory registers a factory function for creating providers.
func (r *FactoryRegistry[T, C]) RegisterFactory(name string, factory func() T) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if factory == nil {
		panic(fmt.Sprintf("RegisterFactory: factory is nil for %s %s", r.providerType, name))
	}
	r.factories[name] = factory
}

// GetFactory retrieves a registered factory by name.
func (r *FactoryRegistry[T, C]) GetFactory(name string) (func() T, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	factory, exists := r.factories[name]
	return factory, exists
}

// ListFactories returns all registered factory names.
func (r *FactoryRegistry[T, C]) ListFactories() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// RegisterConfigTransformer registers a config transformation function.
func (r *FactoryRegistry[T, C]) RegisterConfigTransformer(name string, transformer func(map[string]any) (C, error)) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if transformer == nil {
		panic(fmt.Sprintf("RegisterConfigTransformer: transformer is nil for %s %s", r.providerType, name))
	}
	r.configTransforms[name] = transformer
}

// GetConfigTransformer retrieves a registered config transformer.
func (r *FactoryRegistry[T, C]) GetConfigTransformer(name string) (func(map[string]any) (C, error), bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	transformer, exists := r.configTransforms[name]
	return transformer, exists
}

// TransformConfig transforms raw config using the registered transformer.
func (r *FactoryRegistry[T, C]) TransformConfig(name string, rawConfig map[string]any) (C, error) {
	transformer, exists := r.GetConfigTransformer(name)
	if !exists {
		var zero C
		return zero, fmt.Errorf("no config transformer registered for %s provider: %s", r.providerType, name)
	}
	return transformer(rawConfig)
}

// RegisterBuildFromEnv registers a self-configuration function.
func (r *FactoryRegistry[T, C]) RegisterBuildFromEnv(name string, builder func() (T, error)) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if builder == nil {
		panic(fmt.Sprintf("RegisterBuildFromEnv: builder is nil for %s %s", r.providerType, name))
	}
	r.buildFromEnv[name] = builder
}

// GetBuildFromEnv retrieves a registered BuildFromEnv function.
func (r *FactoryRegistry[T, C]) GetBuildFromEnv(name string) (func() (T, error), bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	builder, exists := r.buildFromEnv[name]
	return builder, exists
}

// BuildFromEnv creates a provider using its registered BuildFromEnv function.
func (r *FactoryRegistry[T, C]) BuildFromEnv(name string) (T, error) {
	builder, exists := r.GetBuildFromEnv(name)
	if !exists {
		var zero T
		return zero, fmt.Errorf("no BuildFromEnv registered for %s provider: %s (available: %v)",
			r.providerType, name, r.ListBuildFromEnv())
	}
	return builder()
}

// ListBuildFromEnv returns all registered BuildFromEnv names.
func (r *FactoryRegistry[T, C]) ListBuildFromEnv() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	names := make([]string, 0, len(r.buildFromEnv))
	for name := range r.buildFromEnv {
		names = append(names, name)
	}
	return names
}
