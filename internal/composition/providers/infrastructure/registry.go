package infrastructure

import (
	"fmt"
	"sync"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
)

// Registry manages infrastructure provider instances (database, auth, storage, id)
type Registry struct {
	mu sync.RWMutex

	database contracts.Provider
	auth     contracts.Provider
	storage  contracts.Provider
	id       contracts.Provider
}

// NewRegistry creates a new infrastructure provider registry
func NewRegistry() *Registry {
	return &Registry{}
}

// InitializeAll creates and initializes all infrastructure providers from environment.
// Each provider reads its own configuration from environment variables.
func (r *Registry) InitializeAll() error {
	// Initialize database provider
	dbProvider, err := CreateDatabaseProvider()
	if err != nil {
		return fmt.Errorf("failed to create database provider: %w", err)
	}
	r.SetDatabase(dbProvider)

	// Initialize auth provider
	authProvider, err := CreateAuthProvider()
	if err != nil {
		return fmt.Errorf("failed to create auth provider: %w", err)
	}
	r.SetAuth(authProvider)

	// Initialize storage provider
	storageProvider, err := CreateStorageProvider()
	if err != nil {
		return fmt.Errorf("failed to create storage provider: %w", err)
	}
	r.SetStorage(storageProvider)

	// Initialize ID provider
	idProvider, err := CreateIDProvider()
	if err != nil {
		return fmt.Errorf("failed to create ID provider: %w", err)
	}
	r.SetID(idProvider)

	return nil
}

// SetDatabase sets the database provider
func (r *Registry) SetDatabase(provider contracts.Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.database = provider
}

// GetDatabase returns the database provider
func (r *Registry) GetDatabase() contracts.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.database
}

// SetAuth sets the auth provider
func (r *Registry) SetAuth(provider contracts.Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.auth = provider
}

// GetAuth returns the auth provider
func (r *Registry) GetAuth() contracts.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.auth
}

// SetStorage sets the storage provider
func (r *Registry) SetStorage(provider contracts.Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.storage = provider
}

// GetStorage returns the storage provider
func (r *Registry) GetStorage() contracts.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.storage
}

// SetID sets the ID provider
func (r *Registry) SetID(provider contracts.Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.id = provider
}

// GetID returns the ID provider
func (r *Registry) GetID() contracts.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.id
}

// GetAll returns all infrastructure providers as a slice
func (r *Registry) GetAll() []contracts.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var providers []contracts.Provider
	if r.database != nil {
		providers = append(providers, r.database)
	}
	if r.auth != nil {
		providers = append(providers, r.auth)
	}
	if r.storage != nil {
		providers = append(providers, r.storage)
	}
	if r.id != nil {
		providers = append(providers, r.id)
	}
	return providers
}

// Close closes all infrastructure providers
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error

	if r.database != nil {
		if err := r.database.Close(); err != nil {
			errs = append(errs, fmt.Errorf("database: %w", err))
		}
	}
	if r.auth != nil {
		if err := r.auth.Close(); err != nil {
			errs = append(errs, fmt.Errorf("auth: %w", err))
		}
	}
	if r.storage != nil {
		if err := r.storage.Close(); err != nil {
			errs = append(errs, fmt.Errorf("storage: %w", err))
		}
	}
	if r.id != nil {
		if err := r.id.Close(); err != nil {
			errs = append(errs, fmt.Errorf("id: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing infrastructure providers: %v", errs)
	}
	return nil
}
