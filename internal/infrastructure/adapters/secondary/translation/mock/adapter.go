//go:build mock_db

package mock

import (
	"context"
	"fmt"
	"strings"

	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterTranslationProvider(
		"mock",
		func() ports.TranslationService {
			return NewMockTranslationService()
		},
		nil, // No config transformer needed
	)
	registry.RegisterTranslationBuildFromEnv("mock", buildFromEnv)
}

// buildFromEnv creates a mock translation service.
func buildFromEnv() (ports.TranslationService, error) {
	return NewMockTranslationService(), nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// MockTranslationService provides a simple mock translation service for testing.
// It returns the key with parameter substitution for debugging purposes.
type MockTranslationService struct{}

// NewMockTranslationService creates a new mock translation service.
func NewMockTranslationService() *MockTranslationService {
	return &MockTranslationService{}
}

// Get returns the key with parameters substituted for debugging.
func (s *MockTranslationService) Get(ctx context.Context, businessType, key string, params ...any) string {
	return s.formatMessage(key, params...)
}

// GetWithDefault returns the default message with parameters substituted.
func (s *MockTranslationService) GetWithDefault(ctx context.Context, businessType, key, defaultMessage string, params ...any) string {
	return s.formatMessage(defaultMessage, params...)
}

// formatMessage performs simple parameter substitution.
func (s *MockTranslationService) formatMessage(message string, params ...any) string {
	if len(params) == 0 || params[0] == nil {
		return message
	}

	// Assuming params[0] is a map[string]any for named parameters
	if paramMap, ok := params[0].(map[string]any); ok {
		for k, v := range paramMap {
			placeholder := fmt.Sprintf("{%s}", k)
			message = strings.ReplaceAll(message, placeholder, fmt.Sprintf("%v", v))
		}
	}

	return message
}

// Name returns the adapter name.
func (s *MockTranslationService) Name() string {
	return "mock"
}

// IsEnabled returns whether the adapter is enabled.
func (s *MockTranslationService) IsEnabled() bool {
	return true
}

var _ ports.TranslationService = (*MockTranslationService)(nil)
