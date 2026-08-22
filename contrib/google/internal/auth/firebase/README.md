Firebase auth adapter (espyna auth port implementation)

This package is Espyna's Firebase secondary auth adapter. The enclosing Google contribution
registers it through `contrib/google/register_firebase.go` in Firebase/Google-tagged builds.

The concrete adapter satisfies the provider/service surfaces consumed by Espyna:

- `ports.AuthProvider` via `FirebaseAuthAdapter`
- `ports.AuthService` (user lifecycle, token verification, capability queries, password/token operations)

Key files:

- `adapter.go` — initialization/health/close, `VerifyToken`, token-verification hardening,
  provider-specific user management, and custom-token minting for an existing email or UID.
- `client_manager.go` — creates/caches Firebase app and auth clients and holds project credentials loaded by GCP helpers.
- `client.go` — thin auth-client wrapper over Firebase SDK operations.

Config and wiring:

- `AUTH_FIREBASE_PROJECT_ID` is read at build-from-env initialization.
- The adapter exposes provider name `firebase` and gates all operations on `Initialize` + enabled state.
- It is mounted as a secondary adapter, selected through `ports`/`registry` rather than direct Firebase imports in higher layers.
- `CreateCustomToken` resolves an existing Firebase identity and delegates to Admin SDK
  `CustomToken`; it never logs or persists the signed token.

Testing:

- `go test -short ./internal/auth/firebase` from `contrib/google` compiles and exercises the
  provider package without requiring a live project.
