// Package gcp is the concern-agnostic GCP credential primitive shared by every
// Google-backed concern (AUTH/firebase, STORAGE/gcs, DATABASE/firestore, ...).
//
// Charter — this package:
//   - Holds the credential SHAPE (CredentialConfig) and the option-builder
//     (GetClientOption: CredentialConfig -> google api option.ClientOption).
//   - Reads the environment ONLY via LoadCredentialConfig(prefix, projectID),
//     where the CALLER injects its fully-explicit {CONCERN}_{PROVIDER}_ prefix
//     and its already-resolved resource project.
//   - MUST NOT hardcode any {CONCERN}_{PROVIDER}_ literal, MUST NOT read any
//     global/shared env name (no bare GOOGLE_APPLICATION_CREDENTIALS), and MUST
//     NOT os.Setenv. Each concern passes its own scoped credentials directly to
//     its SDK client, so AUTH and STORAGE can target entirely different GCP
//     projects/credentials with no shared state.
//
// Authentication methods (resolved by GetClientOption):
//   1. Service-account JSON file at {prefix}CREDENTIALS_FILE (local/non-managed escape hatch)
//   2. Application Default Credentials (ADC; managed-runtime default)
//
// Inline {prefix}SA_*, {prefix}USE_SERVICE_ACCOUNT, and the key-path alias are
// retired and fail closed by environment name before SDK construction.
//
// Usage (the caller — a concern adapter — owns the prefix):
//
//	cfg, err := gcp.LoadCredentialConfig("STORAGE_GCS_", explicitProjectID)
//	opt, err := gcp.GetClientOption(cfg)
//	if err != nil {
//	    return err
//	}
//	client, err := storage.NewClient(ctx, opt)
//
// The package uses build tag "google" so it compiles only into Google-enabled
// binaries, keeping non-Google builds lean.
package gcp
