//go:build aws && s3

package consumer

// NOTE: storage/s3 adapter does not yet have an init() with
// registry.RegisterStorageProvider. This import is ready for when
// self-registration is added to that package.
import _ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/storage/s3"
