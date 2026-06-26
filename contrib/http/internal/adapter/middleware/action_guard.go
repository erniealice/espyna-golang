//go:build http

// action_guard.go
//
// ActionWorkspaceGuardMiddleware closes the form/session workspace_id
// cross-validation gap: browser cookies are per-domain, not per-tab. After
// URL-driven session rotation, a form rendered on /w/A/... can be submitted
// with the post-rotation /w/B/... cookie, and the action handler would silently
// execute the mutation in workspace B.
//
// The middleware sits IN FRONT of every /action/* handler. For unsafe methods
// (POST/PUT/PATCH/DELETE), it requires two form fields:
//
//	_workspace_id      -- the workspace_id captured at form-render time
//	_workspace_id_sig  -- HMAC over (_workspace_id + action_path + nonce)
//
// The middleware verifies BOTH:
//   - the signature is valid for this action path
//   - _workspace_id matches consumer.GetWorkspaceIDFromContext(ctx)
//
// On mismatch: 409 Conflict, HX-Refresh: true for HTMX clients.
package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// --- Sentinel errors ---

var (
	// ErrMissingWorkspaceField is returned when the form payload is missing
	// either _workspace_id or _workspace_id_sig.
	ErrMissingWorkspaceField = errors.New("action_workspace_guard: missing _workspace_id / _workspace_id_sig field")

	// ErrInvalidSignature is returned when the HMAC over the form values
	// fails verification.
	ErrInvalidSignature = errors.New("action_workspace_guard: invalid signature")

	// ErrWorkspaceMismatch is returned when the form's _workspace_id does
	// not match the session's current workspace_id.
	ErrWorkspaceMismatch = errors.New("action_workspace_guard: form/session workspace mismatch")
)

// --- Constants ---

const (
	// FormFieldWorkspaceID is the form key for the workspace_id captured at
	// render time.
	FormFieldWorkspaceID = "_workspace_id"

	// FormFieldWorkspaceIDSig is the form key for the HMAC signature.
	FormFieldWorkspaceIDSig = "_workspace_id_sig"

	// EnvKeyWorkspaceFormHMAC is the canonical env var for the action
	// guard's HMAC secret.
	EnvKeyWorkspaceFormHMAC = "SECURITY_WORKSPACEFORM_HMAC_KEY"

	// EnvKeyFallbackHMAC is the fallback env var for dev/test.
	EnvKeyFallbackHMAC = "AUTH_PASSWORD_RESET_TOKEN_SECRET"

	// nonceBytes is the per-form nonce length in bytes. 16 bytes = 128 bits.
	nonceBytes = 16
)

// --- WorkspaceFormSigner: render-side helper ---

// WorkspaceFormSigner produces the (value, signature) pair that the
// template helper writes into hidden <input> elements at form-render time.
// It also verifies signatures on the action side.
//
// Construct one per process via NewWorkspaceFormSigner. Concurrent-safe.
type WorkspaceFormSigner struct {
	key []byte
}

// NewWorkspaceFormSigner constructs a signer from a raw secret. The key
// MUST be non-empty; callers should resolve via SecretFromEnv.
//
// Panics if key is empty.
func NewWorkspaceFormSigner(key string) *WorkspaceFormSigner {
	if key == "" {
		panic("action_workspace_guard: HMAC key required (set SECURITY_WORKSPACEFORM_HMAC_KEY)")
	}
	return &WorkspaceFormSigner{key: []byte(key)}
}

// SecretFromEnv resolves the HMAC secret from the environment, applying
// the fallback chain (SECURITY_WORKSPACEFORM_HMAC_KEY -> AUTH_PASSWORD_RESET_TOKEN_SECRET).
// Returns "" when neither var is set.
func SecretFromEnv(getenv func(string) string) string {
	if v := getenv(EnvKeyWorkspaceFormHMAC); v != "" {
		return v
	}
	if v := getenv(EnvKeyFallbackHMAC); v != "" {
		return v
	}
	return ""
}

// SignFields signs (workspaceID, actionPath) with a fresh nonce and
// returns the value of the _workspace_id_sig form field. The format is
// base64url(nonce) + "." + base64url(hmac_sha256).
func (s *WorkspaceFormSigner) SignFields(workspaceID, actionPath string) (sigValue string, err error) {
	if workspaceID == "" {
		return "", fmt.Errorf("action_workspace_guard: workspaceID required for signing")
	}
	nonce := make([]byte, nonceBytes)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("action_workspace_guard: nonce generation failed: %w", err)
	}
	nonceB64 := base64.RawURLEncoding.EncodeToString(nonce)
	mac := s.mac(workspaceID, actionPath, nonceB64)
	sigB64 := base64.RawURLEncoding.EncodeToString(mac)
	return nonceB64 + "." + sigB64, nil
}

// Verify checks that signatureValue is a valid HMAC over
// (workspaceID, actionPath, nonce).
func (s *WorkspaceFormSigner) Verify(workspaceID, actionPath, signatureValue string) error {
	if signatureValue == "" || workspaceID == "" || actionPath == "" {
		return ErrInvalidSignature
	}
	idx := strings.LastIndex(signatureValue, ".")
	if idx <= 0 || idx == len(signatureValue)-1 {
		return ErrInvalidSignature
	}
	nonceB64 := signatureValue[:idx]
	sigB64 := signatureValue[idx+1:]
	providedSig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return ErrInvalidSignature
	}
	expectedSig := s.mac(workspaceID, actionPath, nonceB64)
	if !hmac.Equal(providedSig, expectedSig) {
		return ErrInvalidSignature
	}
	return nil
}

// mac produces the raw HMAC-SHA256 over the canonical payload.
func (s *WorkspaceFormSigner) mac(workspaceID, actionPath, nonceB64 string) []byte {
	h := hmac.New(sha256.New, s.key)
	h.Write([]byte(workspaceID))
	h.Write([]byte("|"))
	h.Write([]byte(actionPath))
	h.Write([]byte("|"))
	h.Write([]byte(nonceB64))
	return h.Sum(nil)
}

// --- Middleware ---

// ActionWorkspaceGuardConfig wires the middleware. Signer is required.
type ActionWorkspaceGuardConfig struct {
	// Signer verifies the _workspace_id_sig form field. Required.
	Signer *WorkspaceFormSigner

	// SessionWorkspaceID reads the session's current workspace_id from
	// the request context. Required. Typically wired to
	// consumer.GetWorkspaceIDFromContext.
	SessionWorkspaceID func(ctx context.Context) string

	// PathPrefix scopes the middleware. Defaults to "/action/".
	PathPrefix string

	// SafeMethods are HTTP methods that bypass the guard. Defaults to
	// GET/HEAD/OPTIONS.
	SafeMethods map[string]bool
}

// ActionWorkspaceGuardMiddleware enforces the form/session workspace_id
// invariant on /action/* mutating requests.
type ActionWorkspaceGuardMiddleware struct {
	cfg ActionWorkspaceGuardConfig
}

// NewActionWorkspaceGuardMiddleware constructs the middleware. Panics if
// cfg.Signer is nil.
func NewActionWorkspaceGuardMiddleware(cfg ActionWorkspaceGuardConfig) func(next http.Handler) http.Handler {
	if cfg.Signer == nil {
		panic("action_workspace_guard middleware: Signer is required")
	}
	if cfg.SessionWorkspaceID == nil {
		// No-op default: returns "" which makes the guard a pass-through
		// (pre-workspace actions). Callers should wire this to
		// consumer.GetWorkspaceIDFromContext.
		cfg.SessionWorkspaceID = func(context.Context) string { return "" }
	}
	if cfg.PathPrefix == "" {
		cfg.PathPrefix = "/action/"
	}
	if cfg.SafeMethods == nil {
		cfg.SafeMethods = map[string]bool{
			http.MethodGet:     true,
			http.MethodHead:    true,
			http.MethodOptions: true,
		}
	}
	m := &ActionWorkspaceGuardMiddleware{cfg: cfg}
	return m.handle
}

func (m *ActionWorkspaceGuardMiddleware) handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 0. Path scope -- only /action/* requests pass through the guard.
		if !strings.HasPrefix(r.URL.Path, m.cfg.PathPrefix) {
			next.ServeHTTP(w, r)
			return
		}

		// 0a. /action/auth/* is the LOGIN FLOW (pre-workspace-binding).
		if strings.HasPrefix(r.URL.Path, m.cfg.PathPrefix+"auth/") {
			next.ServeHTTP(w, r)
			return
		}

		// 0b. /action/admin/switch-workspace is the workspace-switch handler
		//     -- by definition the form's workspace_id is DIFFERENT from the
		//     session's workspace_id.
		if strings.HasPrefix(r.URL.Path, m.cfg.PathPrefix+"admin/switch-workspace") {
			next.ServeHTTP(w, r)
			return
		}

		// 1. Method scope -- safe methods bypass.
		if m.cfg.SafeMethods[r.Method] {
			next.ServeHTTP(w, r)
			return
		}

		// 2. Workspace scope -- actions before workspace binding cannot be
		//    cross-validated.
		sessionWsID := m.cfg.SessionWorkspaceID(r.Context())
		if sessionWsID == "" {
			next.ServeHTTP(w, r)
			return
		}

		// 3. Parse the form.
		var (
			formWsID string
			formSig  string
		)
		ct := r.Header.Get("Content-Type")
		if strings.HasPrefix(ct, "multipart/") {
			if err := r.ParseMultipartForm(32 << 20); err != nil {
				log.Printf("[action_workspace_guard] multipart parse error path=%s: %v",
					r.URL.Path, err)
				m.writeMismatch(w, r, "Failed to parse form")
				return
			}
			formWsID = r.FormValue(FormFieldWorkspaceID)
			formSig = r.FormValue(FormFieldWorkspaceIDSig)
		} else {
			if err := r.ParseForm(); err != nil {
				log.Printf("[action_workspace_guard] form parse error path=%s: %v",
					r.URL.Path, err)
				m.writeMismatch(w, r, "Failed to parse form")
				return
			}
			formWsID = r.PostFormValue(FormFieldWorkspaceID)
			formSig = r.PostFormValue(FormFieldWorkspaceIDSig)
		}

		// 4. Both fields must be present.
		if formWsID == "" || formSig == "" {
			log.Printf("[action_workspace_guard] missing fields path=%s session_ws=%s",
				r.URL.Path, sessionWsID)
			m.writeMismatch(w, r, "Workspace context missing from form; please reload")
			return
		}

		// 5. Signature verification.
		if err := m.cfg.Signer.Verify(formWsID, r.URL.Path, formSig); err != nil {
			log.Printf("[action_workspace_guard] signature invalid path=%s form_ws=%s session_ws=%s: %v",
				r.URL.Path, formWsID, sessionWsID, err)
			m.writeMismatch(w, r, "Workspace signature invalid; please reload")
			return
		}

		// 6. Form/session cross-validation.
		if formWsID != sessionWsID {
			log.Printf("[action_workspace_guard] workspace mismatch path=%s form_ws=%s session_ws=%s",
				r.URL.Path, formWsID, sessionWsID)
			m.writeMismatch(w, r, "Workspace changed, please reload")
			return
		}

		// 7. All checks passed.
		next.ServeHTTP(w, r)
	})
}

func (m *ActionWorkspaceGuardMiddleware) writeMismatch(
	w http.ResponseWriter, r *http.Request, message string,
) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Refresh", "true")
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	fmt.Fprintf(w, `{"error":%q}`, message)
}
