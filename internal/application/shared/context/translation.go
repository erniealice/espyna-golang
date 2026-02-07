package context

import (
	"context"
	"leapfor.xyz/espyna/internal/application/ports"
	"strings"
)

// TranslationHelper provides translation functionality for use cases.
type TranslationHelper struct {
	translationService ports.TranslationService
}

// NewTranslationHelper creates a new translation helper.
func NewTranslationHelper(translationService ports.TranslationService) *TranslationHelper {
	return &TranslationHelper{
		translationService: translationService,
	}
}

// GetTranslatedMessage retrieves translated message with fallback.
// This is a centralized version of the getTranslatedMessage method found in many use cases.
func (th *TranslationHelper) GetTranslatedMessage(ctx context.Context, businessType, key, fallback string) string {
	if th.translationService != nil {
		return th.translationService.GetWithDefault(ctx, businessType, key, fallback)
	}
	return fallback
}

// GetTranslatedMessageWithContext is a convenience method that extracts business type from context.
func (th *TranslationHelper) GetTranslatedMessageWithContext(ctx context.Context, key, fallback string) string {
	businessType := ExtractBusinessTypeFromContext(ctx)
	return th.GetTranslatedMessage(ctx, businessType, key, fallback)
}

// Static helper functions for use cases that already have translation service in their struct

// GetTranslatedMessage is a static helper for use cases that inject their own translation service.
func GetTranslatedMessage(ctx context.Context, translationService ports.TranslationService, businessType, key, fallback string) string {
	if translationService != nil {
		return translationService.GetWithDefault(ctx, businessType, key, fallback)
	}
	return fallback
}

// GetTranslatedMessageWithContext is a static helper that extracts business type from context.
func GetTranslatedMessageWithContext(ctx context.Context, translationService ports.TranslationService, key, fallback string) string {
	businessType := ExtractBusinessTypeFromContext(ctx)
	return GetTranslatedMessage(ctx, translationService, businessType, key, fallback)
}

// GetTranslatedMessageWithContextAndTags is a static helper that extracts business type from context and applies tags.
func GetTranslatedMessageWithContextAndTags(ctx context.Context, translationService ports.TranslationService, key string, tags map[string]interface{}, fallback string) string {
	businessType := ExtractBusinessTypeFromContext(ctx)
	message := GetTranslatedMessage(ctx, translationService, businessType, key, fallback)

	// Apply tags by replacing placeholders in the message
	for tagKey, tagValue := range tags {
		placeholder := "{" + tagKey + "}"
		message = strings.ReplaceAll(message, placeholder, toString(tagValue))
	}

	return message
}

// Contains checks if a string contains a substring (case insensitive)
func Contains(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}

// toString converts an interface{} to string
func toString(value interface{}) string {
	if value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}
