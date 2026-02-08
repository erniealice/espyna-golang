package file

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterTranslationProvider(
		"file",
		func() ports.TranslationService {
			return NewFileTranslationAdapter()
		},
		transformConfig,
	)
	registry.RegisterTranslationBuildFromEnv("file", buildFromEnv)
}

// buildFromEnv creates and initializes a file-based translation service from environment variables.
func buildFromEnv() (ports.TranslationService, error) {
	translationsPath := os.Getenv("LEAPFOR_TRANSLATION_PATH")
	if translationsPath == "" {
		translationsPath = os.Getenv("TRANSLATIONS_PATH")
	}
	if translationsPath == "" {
		translationsPath = "translations" // Default path
	}

	adapter := NewFileTranslationAdapter()
	config := &registry.TranslationProviderConfig{
		Provider:         "file",
		Enabled:          true,
		TranslationsPath: translationsPath,
	}

	if err := adapter.Initialize(config); err != nil {
		return nil, fmt.Errorf("file translation: failed to initialize: %w", err)
	}
	return adapter, nil
}

// transformConfig converts raw config map to TranslationProviderConfig.
func transformConfig(rawConfig map[string]any) (*registry.TranslationProviderConfig, error) {
	config := &registry.TranslationProviderConfig{
		Provider: "file",
		Enabled:  true,
	}

	if path, ok := rawConfig["translations_path"].(string); ok && path != "" {
		config.TranslationsPath = path
	} else {
		config.TranslationsPath = "translations"
	}

	return config, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// FileTranslationAdapter implements ports.TranslationService using JSON files.
type FileTranslationAdapter struct {
	translationsPath string
	cache            map[string]map[string]string // locale_businessType -> key -> message
	mutex            sync.RWMutex
	enabled          bool
}

// NewFileTranslationAdapter creates a new file-based translation adapter.
func NewFileTranslationAdapter() *FileTranslationAdapter {
	return &FileTranslationAdapter{
		cache:   make(map[string]map[string]string),
		enabled: false,
	}
}

// Initialize sets up the file translation adapter with configuration.
func (a *FileTranslationAdapter) Initialize(config *registry.TranslationProviderConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is required")
	}

	a.translationsPath = config.TranslationsPath
	if a.translationsPath == "" {
		a.translationsPath = "translations"
	}

	a.enabled = config.Enabled
	log.Printf("✅ File translation provider initialized (path: %s)", a.translationsPath)
	return nil
}

// Get retrieves a translated message for a given business type and key.
func (a *FileTranslationAdapter) Get(ctx context.Context, businessType, key string, params ...any) string {
	if !a.enabled {
		return key
	}

	// Load messages for the specific business type
	messages, err := a.loadMessages("en", businessType)
	if err != nil {
		// Fallback to general if specific business type messages fail to load
		messages, _ = a.loadMessages("en", "general")
	}

	translated, ok := messages[key]
	if !ok {
		// If not found in business-specific, try general fallback
		if businessType != "general" {
			generalMessages, err := a.loadMessages("en", "general")
			if err != nil {
				return key
			}
			translated, ok = generalMessages[key]
		}
		if !ok {
			return key // Return the key itself if no translation is found
		}
	}

	return a.formatMessage(translated, params...)
}

// GetWithDefault retrieves a translated message with a fallback to a default message.
func (a *FileTranslationAdapter) GetWithDefault(ctx context.Context, businessType, key, defaultMessage string, params ...any) string {
	translated := a.Get(ctx, businessType, key, params...)
	if translated == key {
		return a.formatMessage(defaultMessage, params...)
	}
	return translated
}

// formatMessage performs simple parameter substitution.
func (a *FileTranslationAdapter) formatMessage(message string, params ...any) string {
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

// loadMessages loads and merges translation messages for a given locale and business type.
func (a *FileTranslationAdapter) loadMessages(locale, businessType string) (map[string]string, error) {
	cacheKey := fmt.Sprintf("%s_%s", locale, businessType)

	a.mutex.RLock()
	if messages, ok := a.cache[cacheKey]; ok {
		a.mutex.RUnlock()
		return messages, nil
	}
	a.mutex.RUnlock()

	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Double-check after acquiring lock
	if messages, ok := a.cache[cacheKey]; ok {
		return messages, nil
	}

	mergedMessages := make(map[string]string)

	// 1. Load general translations (if not already business-specific)
	if businessType != "general" {
		generalPath := filepath.Join(a.translationsPath, locale, "general")
		if err := a.loadDirectory(generalPath, mergedMessages); err != nil {
			// General translations are optional
		}
	}

	// 2. Load business-specific translations
	businessPath := filepath.Join(a.translationsPath, locale, businessType)
	if err := a.loadDirectory(businessPath, mergedMessages); err != nil {
		return nil, fmt.Errorf("failed to load translations for %s/%s: %w", locale, businessType, err)
	}

	a.cache[cacheKey] = mergedMessages
	return mergedMessages, nil
}

// loadDirectory reads all JSON files from a directory and merges them into the provided map.
func (a *FileTranslationAdapter) loadDirectory(dirPath string, targetMap map[string]string) error {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return err // Directory might not exist
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(dirPath, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}

		var messages map[string]any
		if err := json.Unmarshal(data, &messages); err != nil {
			return fmt.Errorf("failed to unmarshal JSON from %s: %w", filePath, err)
		}

		// Flatten and merge messages
		a.flattenAndMerge(messages, "", targetMap)
	}

	return nil
}

// flattenAndMerge recursively flattens a nested map and merges it into the target map.
func (a *FileTranslationAdapter) flattenAndMerge(source map[string]any, prefix string, target map[string]string) {
	for key, value := range source {
		newKey := key
		if prefix != "" {
			newKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]any:
			a.flattenAndMerge(v, newKey, target)
		case string:
			target[newKey] = v
		default:
			target[newKey] = fmt.Sprintf("%v", v)
		}
	}
}

// Name returns the adapter name.
func (a *FileTranslationAdapter) Name() string {
	return "file"
}

// IsEnabled returns whether the adapter is enabled.
func (a *FileTranslationAdapter) IsEnabled() bool {
	return a.enabled
}

var _ ports.TranslationService = (*FileTranslationAdapter)(nil)
