package infrastructure

import (
	"fmt"
	"time"
)

// IDService provides ID generation functionality for the application
// This interface abstracts ID generation to support different implementations
type IDService interface {
	// GenerateID creates a new unique identifier
	GenerateID() string

	// GenerateIDWithPrefix creates a unique identifier with a specified prefix
	// Useful for maintaining readable ID formats (e.g., "client_uuid", "admin_uuid")
	GenerateIDWithPrefix(prefix string) string

	// IsEnabled returns whether ID service is available and enabled
	IsEnabled() bool

	// GetProviderInfo returns information about the underlying ID provider
	GetProviderInfo() string
}

// NewNoOpIDService creates a fallback ID service for testing/compatibility
func NewNoOpIDService() IDService {
	return &NoOpIDService{}
}

// NoOpIDService provides fallback functionality when no ID service is configured
type NoOpIDService struct{}

func (s *NoOpIDService) GenerateID() string {
	return fmt.Sprintf("noop_%d", time.Now().UnixNano())
}

func (s *NoOpIDService) GenerateIDWithPrefix(prefix string) string {
	return fmt.Sprintf("%s_noop_%d", prefix, time.Now().UnixNano())
}

func (s *NoOpIDService) IsEnabled() bool {
	return false
}

func (s *NoOpIDService) GetProviderInfo() string {
	return "NoOp ID Service (fallback)"
}
