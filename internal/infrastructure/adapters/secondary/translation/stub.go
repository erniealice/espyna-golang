//go:build mock_db

// Package translation provides backwards-compatible factory functions for translation services.
// This stub file exists for tests that use the old import path.
// New code should import the specific adapter packages:
//   - translation/file
//   - translation/lyngua
//   - translation/noop
//   - translation/mock
package translation

import (
	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/translation/mock"
)

// NewLynguaTranslationService returns a mock translation service for testing.
// This function is for backwards compatibility with existing tests.
// In test mode (mock_db build tag), it returns a mock service that passes through keys.
//
// For actual lyngua support, use the lyngua adapter with the lyngua build tag.
func NewLynguaTranslationService() ports.TranslationService {
	return mock.NewMockTranslationService()
}

// NewFileTranslationService returns a mock translation service for testing.
// This function is for backwards compatibility with existing tests.
func NewFileTranslationService(path string) ports.TranslationService {
	return mock.NewMockTranslationService()
}

// NewMockTranslationService returns a mock translation service for testing.
func NewMockTranslationService() ports.TranslationService {
	return mock.NewMockTranslationService()
}
