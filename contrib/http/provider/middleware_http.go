//go:build http

// middleware_http.go
//
// Framework-native middleware BRIDGE for the net/http server provider.
//
// This file is the chain-assembly seam mandated by the LOCKED contrib pattern
// (docs/plan/20260614-composition-model-a/contrib-pattern.md §1 C1/C2, §6): the
// real net/http middleware impls live under contrib/http/internal/adapter/
// middleware (Go-`internal/` visibility means ONLY packages rooted at
// contrib/http may import them), so the code that constructs them from the
// framework-AGNOSTIC consumer/http/middleware config MUST live here, in
// package espynahttp (which is rooted at contrib/http).
//
// It is //go:build http so it compiles into the binary exactly when
// CONFIG_SERVER_PROVIDER=http selects the net/http provider (scripts/build.sh
// passes the value verbatim as a build tag). consumer/http reaches these
// builders through a build-tagged dispatcher; this package must NOT import
// consumer or consumer/http (that would close the consumer -> contrib/http
// import cycle), so every dependency arrives as a closure on the agnostic
// config struct.
package provider

import (
	"context"
	"errors"
	"net/http"

	consumermw "github.com/erniealice/espyna-golang/consumer/http/middleware"
	cmw "github.com/erniealice/espyna-golang/contrib/http/internal/adapter/middleware"
)

// BuildWorkspacePath constructs the real net/http WorkspacePath middleware from
// the agnostic consumer/http/middleware config, bridging the neutral
// WorkspaceBinding type to the impl's *Principal and the int32 kind to the
// impl's PrincipalType. It preserves every security invariant of the impl:
// ErrAmbiguousBinding -> picker (no auto-elect), URL-canonical workspace_id
// pinned before guards, rotation rate limit, Strict session cookie, fresh CSRF
// cookie on rotation.
//
// Returns a pass-through when SessionLookup is nil (boot-time stub configs).
// When SessionLookup is set but the auth use cases are unavailable (degraded
// boot — e.g. DB down), BindingResolver / ExecuteSwitch arrive nil; rather than
// panic (NewWorkspacePathMiddleware's documented contract) OR fail OPEN, we
// substitute fail-CLOSED defaults that return ErrNoBinding, so any /w/* request
// collapses to the unified not-found/picker response and NEVER reaches the
// downstream handler. This matches the pre-migration wiring (5570bb1), which
// likewise always supplied non-nil closures that fail closed internally.
func BuildWorkspacePath(cfg consumermw.WorkspacePathConfig) func(http.Handler) http.Handler {
	if cfg.SessionLookup == nil {
		return func(next http.Handler) http.Handler { return next }
	}

	implCfg := cmw.WorkspacePathConfig{
		SlugLookup:              cfg.SlugLookup,
		SessionLookup:           cmw.SessionLookupFunc(cfg.SessionLookup),
		SetCSRFCookie:           cfg.SetCSRFCookie,
		SetSessionCookie:        cfg.SetSessionCookie,
		WithWorkspaceID:         cfg.WithWorkspaceID,
		IsReservedSlug:          cfg.IsReservedSlug,
		AppOrigin:               cfg.AppOrigin,
		SlugCacheTTL:            cfg.SlugCacheTTL,
		RotationRateLimitPerMin: cfg.RotationRateLimitPerMin,
	}

	if cfg.BindingResolver != nil {
		resolve := cfg.BindingResolver
		implCfg.BindingResolver = func(
			ctx context.Context,
			userID, workspaceID string,
			sessionPrincipalKind cmw.PrincipalType,
			sessionPrincipalID string,
		) (*cmw.Principal, error) {
			b, err := resolve(ctx, userID, workspaceID, int32(sessionPrincipalKind), sessionPrincipalID)
			if err != nil {
				return nil, mapWorkspaceBindingErr(err)
			}
			return bindingToPrincipal(b), nil
		}
	} else {
		// Fail-CLOSED default (degraded boot): no resolver → no binding →
		// unified not-found. Never reaches downstream, never panics.
		implCfg.BindingResolver = func(context.Context, string, string, cmw.PrincipalType, string) (*cmw.Principal, error) {
			return nil, cmw.ErrNoBindingInWorkspace
		}
	}

	if cfg.PrincipalLookup != nil {
		lookup := cfg.PrincipalLookup
		implCfg.PrincipalLookup = func(r *http.Request) (cmw.PrincipalType, string) {
			kind, id := lookup(r)
			return cmw.PrincipalType(kind), id
		}
	}

	if cfg.ExecuteSwitch != nil {
		exec := cfg.ExecuteSwitch
		implCfg.ExecuteSwitch = func(
			ctx context.Context,
			userID, token string,
			target *cmw.Principal,
			urlActingAs string,
			requestURL, referer, secFetchSite, userAgent string,
		) (*cmw.WorkspaceSwitchResult, error) {
			res, err := exec(ctx, userID, token, principalToBinding(target), urlActingAs, requestURL, referer, secFetchSite, userAgent)
			if err != nil {
				return nil, err
			}
			if res == nil {
				return nil, nil
			}
			return &cmw.WorkspaceSwitchResult{NewToken: res.NewToken, RedirectURL: res.RedirectURL}, nil
		}
	} else {
		// Fail-CLOSED default (degraded boot): no switch primitive → the
		// rotation fails and the impl redirects to the picker. Never panics.
		implCfg.ExecuteSwitch = func(context.Context, string, string, *cmw.Principal, string, string, string, string, string) (*cmw.WorkspaceSwitchResult, error) {
			return nil, cmw.ErrNoBindingInWorkspace
		}
	}

	return cmw.NewWorkspacePathMiddleware(implCfg)
}

// BuildCSRF constructs the real net/http workspace-claim CSRF middleware from
// the agnostic config. Validates the double-submit token + workspace/session
// claim on /action/* mutations and refreshes the cookie on GET. Returns a
// pass-through when Secret is empty (claim validation disabled).
func BuildCSRF(cfg consumermw.CSRFConfig) func(http.Handler) http.Handler {
	if len(cfg.Secret) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	return cmw.NewWorkspaceCSRFMiddleware(cmw.WorkspaceCSRFConfig{
		Secret:       cfg.Secret,
		SessionToken: cfg.SessionToken,
		WorkspaceID:  cfg.WorkspaceID,
		PathPrefix:   cfg.PathPrefix,
	})
}

// BuildActionGuard constructs the real net/http /action/* workspace form-guard
// from the agnostic config. The HMAC signer is built here from the raw secret
// (NewWorkspaceFormSigner panics on empty — but BuildActionGuard short-circuits
// to a pass-through first, so the boot panic is reserved for the consumer/http
// fatal guard that runs ahead of this for a real auth provider). Returns a
// pass-through when Secret is empty.
func BuildActionGuard(cfg consumermw.ActionGuardConfig) func(http.Handler) http.Handler {
	if len(cfg.Secret) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	signer := cmw.NewWorkspaceFormSigner(string(cfg.Secret))
	return cmw.NewActionWorkspaceGuardMiddleware(cmw.ActionWorkspaceGuardConfig{
		Signer:             signer,
		SessionWorkspaceID: cfg.SessionWorkspaceID,
		PathPrefix:         cfg.PathPrefix,
	})
}

// IssueWorkspaceCSRFCookie re-exports the impl helper so consumer/http can wire
// the WorkspacePath SetCSRFCookie closure without reaching into the impl's
// internal/ package (which it cannot import). Issues a fresh workspace-claim
// CSRF cookie alongside the rotated session cookie.
func IssueWorkspaceCSRFCookie(w http.ResponseWriter, secret []byte, sessionToken, workspaceID string) string {
	return cmw.IssueWorkspaceCSRFCookie(w, secret, sessionToken, workspaceID)
}

// mapWorkspaceBindingErr translates the agnostic sentinel errors a
// BindingResolver may return into the impl's sentinels, so the impl's
// errors.Is branch selection (ambiguous -> picker, none -> unified-not-found)
// fires correctly. Any other error is passed through unchanged (-> 500).
func mapWorkspaceBindingErr(err error) error {
	switch {
	case errors.Is(err, consumermw.ErrAmbiguousBinding):
		return cmw.ErrAmbiguousBinding
	case errors.Is(err, consumermw.ErrNoBinding):
		return cmw.ErrNoBindingInWorkspace
	default:
		return err
	}
}

// bindingToPrincipal converts the neutral agnostic binding to the impl Principal.
func bindingToPrincipal(b *consumermw.WorkspaceBinding) *cmw.Principal {
	if b == nil {
		return nil
	}
	p := &cmw.Principal{
		Type:        cmw.PrincipalType(b.Kind),
		ID:          b.PrincipalID,
		WorkspaceID: b.WorkspaceID,
	}
	if len(b.ActingAsTargets) > 0 {
		p.ActingAsTargets = make([]cmw.ActingAsTarget, 0, len(b.ActingAsTargets))
		for _, t := range b.ActingAsTargets {
			p.ActingAsTargets = append(p.ActingAsTargets, cmw.ActingAsTarget{
				ID:          t.ID,
				WorkspaceID: t.WorkspaceID,
			})
		}
	}
	return p
}

// principalToBinding converts the impl Principal (carrying URL-derived
// acting-as) back to the neutral binding so ExecuteSwitch sees the exact
// target the impl built.
func principalToBinding(p *cmw.Principal) *consumermw.WorkspaceBinding {
	if p == nil {
		return nil
	}
	b := &consumermw.WorkspaceBinding{
		Kind:        int32(p.Type),
		PrincipalID: p.ID,
		WorkspaceID: p.WorkspaceID,
	}
	if len(p.ActingAsTargets) > 0 {
		b.ActingAsTargets = make([]consumermw.WorkspaceActingAsTarget, 0, len(p.ActingAsTargets))
		for _, t := range p.ActingAsTargets {
			b.ActingAsTargets = append(b.ActingAsTargets, consumermw.WorkspaceActingAsTarget{
				ID:          t.ID,
				WorkspaceID: t.WorkspaceID,
			})
		}
	}
	return b
}
