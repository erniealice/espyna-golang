// Package gcp is the concern-agnostic GCP credential primitive shared by every
// Google-backed concern (AUTH/firebase, STORAGE/gcs, DATABASE/firestore, ...).
//
// Charter — this package:
//   - Holds the credential SHAPE (CredentialConfig) and the option-builder
//     (GetClientOption: CredentialConfig -> google api option.ClientOption).
//   - Reads the environment ONLY via DefaultCredentialConfig(prefix), where the
//     CALLER injects its fully-explicit {CONCERN}_{PROVIDER}_ prefix.
//   - MUST NOT hardcode any {CONCERN}_{PROVIDER}_ literal, MUST NOT read any
//     global/shared env name (no bare GOOGLE_APPLICATION_CREDENTIALS), and MUST
//     NOT os.Setenv. Each concern passes its own scoped credentials directly to
//     its SDK client, so AUTH and STORAGE can target entirely different GCP
//     projects/credentials with no shared state.
//
// Authentication methods (resolved by GetClientOption, in order):
//   1. Inline service-account JSON from {prefix}SA_* vars ({prefix}USE_SERVICE_ACCOUNT=true)
//   2. Service-account JSON file at {prefix}CREDENTIALS_FILE or {prefix}SERVICE_ACCOUNT_KEY_PATH
//   3. Application Default Credentials (ADC)
//
// Usage (the caller — a concern adapter — owns the prefix):
//
//	cfg := gcp.DefaultCredentialConfig("STORAGE_GCS_")
//	opt, err := gcp.GetClientOption(cfg)
//	if err != nil {
//	    return err
//	}
//	client, err := storage.NewClient(ctx, opt)
//
// The package uses build tag "google" so it compiles only into Google-enabled
// binaries, keeping non-Google builds lean.
package gcp
