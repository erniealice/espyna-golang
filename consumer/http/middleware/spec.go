// Package middleware — spec.go (AGNOSTIC surface).
//
// This file defines the opaque, declarative middleware Preset that the consumer
// app selects (StandardAdmin) and the consumer/http Server fills with per-slot
// config closures from its OWN use cases. The FIXED security order is baked in
// and is REALIZED by the build-tag-selected chain assembler
// (contrib/http/provider/chain_http.go) — NOT here. This file holds zero
// //go:build tags and imports no impl/contrib package: it is a pure-stdlib leaf
// carrying agnostic config types + closures only.
//
// W2 (docs/plan/20260614-composition-model-a/w2-plan.md): the Preset + the
// BuildChain seam collapse the W1 per-function dispatch trio
// (buildWorkspacePath/buildCSRF/buildActionGuard) AND the inline 11-middleware
// hand-assembly in server.go into ONE shared seam. The app gets StandardAdmin()
// and cannot reach inside, reorder, or splice between the security layers.
package middleware

// presetKind discriminates which canonical middleware bundle a Preset realizes.
// The ORDER + WHICH-slots contract for each kind lives in the chain assembler;
// this discriminator only names the bundle.
type presetKind int

const (
	// presetKindStandardAdmin is the canonical admin chain:
	//   SecurityHeaders → Gzip → Logger → Recovery → LoginRateLimit →
	//   Session → WorkspacePath → CSRF → ActionGuard → Timezone → businessType → mux
	presetKindStandardAdmin presetKind = iota
)

// Preset is an opaque, declarative middleware bundle. The fixed security order
// is baked in (selected by kind); the app cannot reorder it or splice between
// layers. The chain assembler (contrib/http/provider/chain_http.go, build-tag
// selected) reads it via the package-internal accessors below to construct the
// real chain in the correct order.
//
// Preset is a VALUE type (not a pointer) so the app cannot mutate a shared
// instance; StandardAdmin() returns a fresh one each call. The Server fills the
// per-slot config closures during its finalize step from espyna's own use cases
// + env — the app supplies none of them.
type Preset struct {
	// kind selects which fixed-order bundle the chain assembler realizes.
	kind presetKind

	// security carries the resolved SecurityHeaders config (HSTS / CSP-enforce
	// from env). The SecurityHeaders slot is the outermost in the chain.
	security SecurityHeadersConfig

	// session is the resolved session handler (password / mock / nil). When nil
	// the Session slot is a pass-through (boot-time / no auth provider).
	session SessionHandler

	// workspace carries the agnostic WorkspacePath config (W1 trio) — the
	// SlugLookup / BindingResolver / ExecuteSwitch / PrincipalLookup / SetCSRFCookie
	// closures the Server wires from service.Auth + Entity.Workspace use cases.
	workspace WorkspacePathConfig

	// csrf carries the agnostic workspace-claim CSRF config (W1 trio).
	csrf CSRFConfig

	// actionGuard carries the agnostic /action/* form-guard config (W1 trio).
	actionGuard ActionGuardConfig

	// timezone carries the two agnostic closures the Timezone slot needs
	// (GetUserID + LookupTimezone). The chain assembler translates them into the
	// contrib NewTimezoneMiddleware(uidFn, lookupFn) form.
	timezone TimezoneConfig

	// loginRateLimit reports whether the LoginRateLimit slot is active. Always
	// true for StandardAdmin (the chain builds it from env).
	loginRateLimit bool

	// businessType is the default business type pinned into ctx by the
	// businessType slot (innermost before the mux).
	businessType string

	// cookieSecure is the resolved COOKIE_SECURE policy. The chain prelude calls
	// SetSecureCookies(cookieSecure) ONCE at assembly time, before any cookie
	// writer (CSRF / WorkspacePath rotation) is constructed.
	cookieSecure bool

	// extras are WithMiddleware additions. They wrap OUTSIDE the fixed core only
	// (reverse-applied in the chain tail) — never spliced between security layers.
	extras []MiddlewareFunc
}

// StandardAdmin returns the canonical admin Preset with the fixed security order
// baked in. It declares the SHAPE only; the per-slot config closures are filled
// by the consumer/http Server during its finalize step from espyna's own use
// cases + env. There are no app-facing knobs — env drives each middleware
// internally. A fresh value is returned on each call so the caller cannot mutate
// a shared instance.
func StandardAdmin() Preset {
	return Preset{
		kind:           presetKindStandardAdmin,
		loginRateLimit: true,
	}
}

// ── Package-internal field accessors + setters ──────────────────────────────
//
// The chain assembler lives in another package (contrib/http/provider) and
// therefore needs exported-style access to the unexported fields. Go does not
// allow cross-package access to unexported fields, so the assembler reaches them
// through the exported accessor methods below. Keeping the FIELDS unexported
// preserves opacity for the APP (which only ever holds the Preset value and
// calls StandardAdmin); the accessors expose a read surface for the assembler
// without letting the app reorder or rebuild the chain.

// Kind reports which fixed-order bundle this Preset realizes. For the chain
// assembler's use only.
func (p Preset) Kind() int { return int(p.kind) }

// Security returns the resolved SecurityHeaders config.
func (p Preset) Security() SecurityHeadersConfig { return p.security }

// Session returns the resolved session handler (may be nil → pass-through slot).
func (p Preset) Session() SessionHandler { return p.session }

// Workspace returns the agnostic WorkspacePath config.
func (p Preset) Workspace() WorkspacePathConfig { return p.workspace }

// CSRFConfig returns the agnostic workspace-claim CSRF config.
func (p Preset) CSRFConfig() CSRFConfig { return p.csrf }

// ActionGuardConfig returns the agnostic /action/* form-guard config.
func (p Preset) ActionGuardConfig() ActionGuardConfig { return p.actionGuard }

// Timezone returns the agnostic timezone closures.
func (p Preset) Timezone() TimezoneConfig { return p.timezone }

// LoginRateLimitActive reports whether the LoginRateLimit slot is active.
func (p Preset) LoginRateLimitActive() bool { return p.loginRateLimit }

// BusinessType returns the default business type for the businessType slot.
func (p Preset) BusinessType() string { return p.businessType }

// CookieSecure returns the resolved COOKIE_SECURE policy for the chain prelude.
func (p Preset) CookieSecure() bool { return p.cookieSecure }

// Extras returns the WithMiddleware extras (wrap OUTSIDE the fixed core).
func (p Preset) Extras() []MiddlewareFunc { return p.extras }

// ── Server-side finalize setters (called from consumer/http) ─────────────────
//
// These return a NEW Preset value with one slot filled (value-semantics keep the
// original immutable). The Server chains them in finalizePreset(). They are NOT
// app-facing knobs: the app never imports these; only consumer/http does.

// WithSecurity fills the SecurityHeaders slot config.
func (p Preset) WithSecurity(cfg SecurityHeadersConfig) Preset { p.security = cfg; return p }

// WithSession fills the Session slot handler.
func (p Preset) WithSession(h SessionHandler) Preset { p.session = h; return p }

// WithWorkspace fills the WorkspacePath slot config.
func (p Preset) WithWorkspace(cfg WorkspacePathConfig) Preset { p.workspace = cfg; return p }

// WithCSRF fills the CSRF slot config.
func (p Preset) WithCSRF(cfg CSRFConfig) Preset { p.csrf = cfg; return p }

// WithActionGuard fills the ActionGuard slot config.
func (p Preset) WithActionGuard(cfg ActionGuardConfig) Preset { p.actionGuard = cfg; return p }

// WithTimezone fills the Timezone slot closures.
func (p Preset) WithTimezone(cfg TimezoneConfig) Preset { p.timezone = cfg; return p }

// WithBusinessType fills the businessType slot value.
func (p Preset) WithBusinessType(bt string) Preset { p.businessType = bt; return p }

// WithCookieSecure fills the resolved COOKIE_SECURE policy for the chain prelude.
func (p Preset) WithCookieSecure(v bool) Preset { p.cookieSecure = v; return p }

// WithExtras sets the WithMiddleware extras (wrap OUTSIDE the fixed core).
func (p Preset) WithExtras(extras []MiddlewareFunc) Preset { p.extras = extras; return p }
