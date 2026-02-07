package domain

import "context"

// TranslationService defines the interface for translation operations.
type TranslationService interface {
	// Get retrieves a translated message for a given business type.
	// It falls back to the 'general' business type if a key is not found.
	Get(ctx context.Context, businessType, key string, params ...any) string

	// GetWithDefault retrieves a translated message with a fallback.
	GetWithDefault(ctx context.Context, businessType, key, defaultMessage string, params ...any) string
}

// NoOpTranslationService provides a non-operational fallback.
type noOpTranslationService struct{}

func (s *noOpTranslationService) Get(ctx context.Context, businessType, key string, params ...any) string {
	return key // Return the key for debugging
}

func (s *noOpTranslationService) GetWithDefault(ctx context.Context, businessType, key, defaultMessage string, params ...any) string {
	return defaultMessage
}

func NewNoOpTranslationService() TranslationService {
	return &noOpTranslationService{}
}
