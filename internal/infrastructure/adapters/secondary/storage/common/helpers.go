package common

import (
	"fmt"
	"mime"
	"path/filepath"
)

// GenerateObjectID creates a unique identifier for an object
// This is a common helper used across all storage providers
func GenerateObjectID(containerName, objectKey string) string {
	return fmt.Sprintf("%s/%s", containerName, objectKey)
}

// DetectContentType attempts to detect content type from file extension
// Returns empty string if it cannot be determined
func DetectContentType(path string) string {
	contentType := mime.TypeByExtension(filepath.Ext(path))
	return contentType
}
