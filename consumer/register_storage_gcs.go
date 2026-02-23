//go:build google && gcp_storage

package consumer

// Import triggers gcs adapter's init() which registers with registry.RegisterStorageProvider.
import _ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/storage/gcs"
