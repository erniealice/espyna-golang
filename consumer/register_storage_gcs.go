//go:build google && gcp_storage

package consumer

// NOTE: storage/gcs adapter does not yet have an init() with
// registry.RegisterStorageProvider. This import is ready for when
// self-registration is added to that package.
import _ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/storage/gcs"
