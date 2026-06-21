// Package app is the espyna-owned composition CONTRACT.
//
// It owns AppContext (the shared infrastructure handed to domain blocks during
// composition), AppOption (the functional-option used to compose them), and
// AppUIBundle (the app-supplied UI surface). This is the builder-style
// composition seam: a host builds an *AppContext from its booted infra, then
// applies each AppOption (a `func(*AppContext) error` returned by a domain
// block's Block()/EngineBlock()) in order — the consumer.Server fluent builder
// (NewServer().WithBlocks(...).WithUI(...).Build()) is the canonical driver.
//
// These definitions MOVED here from pyeza/app_context.go (2026-06-15
// composition-layer relocation). pyeza keeps RouteRegistrar (a pyeza/view
// re-export) and the label/render types (CommonLabels/TableLabels/HTMLRenderer);
// AppContext references them via the pyeza import (espyna already depends on
// pyeza — the legal direction). Many infrastructure slots remain `any` so this
// package takes no cycle-forming dependency on the domain-block packages; the
// concretely-typeable slots are typed where the dependency is a leaf.
package app

import (
	"database/sql"

	pyeza "github.com/erniealice/pyeza-golang"
)

// AppOption configures a domain block within an application. It is the
// functional-option that drives the builder: domain packages export
// Block()/EngineBlock() functions returning AppOption, and the host applies
// each in sequence (consumer.Server.WithBlocks → the opt loop in Build()).
type AppOption func(*AppContext) error

// AppContext provides shared infrastructure to domain blocks during composition.
// Typed fields for pyeza-known types (routes, labels, config) + leaf-package
// types; opaque (any) fields for backend infrastructure whose concrete type
// would form an import cycle — domain blocks type-assert those to their expected
// types (e.g., *consumer.Container, *consumer.UseCases).
type AppContext struct {
	// === Composition targets ===

	// Routes is the route registrar for module registration.
	// Domain blocks call module.RegisterRoutes(ctx.Routes).
	Routes pyeza.RouteRegistrar

	// === Shared labels ===

	// Common holds shared UI labels (buttons, pagination, filters, etc.)
	Common pyeza.CommonLabels

	// Table holds shared table/grid labels (columns, actions, etc.)
	Table pyeza.TableLabels

	// === Configuration ===

	// BusinessType is the configured business type (e.g., "professional", "service")
	// Used by domain blocks to load industry-specific route/label overrides from lyngua.
	BusinessType string

	// === Database ===

	// SqlDB is the raw SQL database connection for report queries.
	// May be nil for non-SQL providers.
	SqlDB *sql.DB

	// === Opaque backend infrastructure ===
	// Domain blocks type-assert these to their expected types.

	// Container is the backend infrastructure container.
	// Type-assert to *consumer.Container in domain blocks.
	Container any

	// UseCases is the use cases aggregate from the backend container.
	// Type-assert to the appropriate use cases type in domain blocks.
	UseCases any

	// DB is the database adapter for CRUD operations.
	// Type-assert to centymo.DataSource or similar in domain blocks.
	DB any

	// Translations is the translation provider for loading domain-specific labels/routes.
	// Type-assert to *lynguaV1.TranslationProvider in domain blocks.
	Translations any

	// RefChecker provides reference checking for deletable-state validation.
	// Type-assert to reference.Checker in domain blocks.
	RefChecker any

	// === Shared cross-cutting operations ===
	// These are pre-computed closures from the consumer app's infrastructure.
	// Using function closures avoids importing hybra/esqyma proto types here.

	// UploadFile uploads a file to object storage (for attachments).
	UploadFile any // func(ctx context.Context, bucket, key string, content []byte, contentType string) error

	// DownloadFile fetches a file's bytes from object storage (for preview/download).
	DownloadFile any // func(ctx context.Context, bucket, key string) ([]byte, error)

	// ListAttachments lists attachments for an entity.
	ListAttachments any // typed by consumer — domain blocks type-assert

	// CreateAttachment creates an attachment record.
	CreateAttachment any

	// ReadAttachment reads a single attachment record by ID.
	ReadAttachment any

	// DeleteAttachment deletes an attachment record.
	DeleteAttachment any

	// NewAttachmentID generates a new attachment ID.
	NewAttachmentID any // func() string

	// UploadImage uploads an image to object storage (for product images).
	UploadImage any

	// UploadTemplate uploads a document template to storage.
	UploadTemplate any

	// SendEmail sends an email (for invoices, notifications).
	SendEmail any

	// GenerateDoc generates a document from a template + data.
	GenerateDoc any

	// ListAuditHistory lists audit trail entries for an entity.
	ListAuditHistory any

	// === Document template CRUD ===
	ListDocTemplates  any
	CreateDocTemplate any
	UpdateDocTemplate any
	DeleteDocTemplate any

	// === Entity-specific helpers ===
	// These are app-specific closures that some domain blocks need.

	// GetUsersByRoleID returns users assigned to a role (for role detail page).
	GetUsersByRoleID any

	// GetDashboardData returns dashboard statistics.
	GetDashboardData any

	// HashPassword hashes a password with bcrypt.
	HashPassword any

	// GetUserWorkspacesMap returns workspace chip data per user (for the user list workspace chip column).
	GetUserWorkspacesMap any

	// SecureWorkspaceSwitch is the optional host-provided override for
	// /action/admin/switch-workspace. When non-nil, the entydad workspace
	// block routes the sidebar workspace-switcher through this closure
	// instead of the legacy in-place SwitchWorkspace use case. The
	// service-admin host wires this to its executePrincipalSwitch primitive
	// so the switch rotates the session token, locks the target binding
	// inside tx, and writes an audit row — matching the workspace-boundary
	// rotation invariant (Q-WS-13) and the audit-on-every-switch invariant
	// (red-team A-4 / X-2). A1 fix WKR-P0-1 (2026-05-22).
	//
	// Type-assert to workspaceaction.SecureSwitchFn inside the entydad
	// block. Kept as `any` here to avoid pulling entydad as a dependency.
	SecureWorkspaceSwitch any

	// SecureWorkspaceSwitchResolveUserID extracts the authenticated user_id
	// from the request. Required when SecureWorkspaceSwitch is set.
	// Type-assert to func(r *http.Request) string.
	SecureWorkspaceSwitchResolveUserID any

	// SecureWorkspaceSwitchSetSessionCookie writes the post-rotation
	// session cookie. Required when SecureWorkspaceSwitch is set.
	// Type-assert to func(w http.ResponseWriter, token string).
	SecureWorkspaceSwitchSetSessionCookie any

	// ComposeResult accumulates Nav contributions and RouteMap entries from
	// each per-package Engine.Assemble call. After all domain blocks have run,
	// the host reads this merged Result to build a NavResolver that can resolve
	// sidebar hrefs across all packages. Initialized by the host before blocks
	// run; each block type-asserts to *compose.Result and calls
	// ComposeResult.MergeFrom(blockResult) after its own eng.Assemble
	// returns successfully. Typed as `any` here; the compose package is a
	// sibling (consumer/compose) so this is concretely-typeable in a later cut.
	ComposeResult any

	// AuthDeps carries the pre-assembled auth module dependencies (login,
	// signup, reset-password, change-password, logout, multi-principal
	// chooser). The host's composition layer builds the full deps struct
	// including type bridges and passes it here; the entydad engine block
	// copies it into block.Infra.AuthDeps for the AuthUnit. Typed as `any`
	// to avoid pulling entydad/service/auth as a dependency.
	// Type-assert to *auth.Deps inside the entydad block.
	AuthDeps any

	// === HTTP-adapter finalize slots (Wave B D1) ===

	// UI is the complete app-supplied UI/labels bundle (*AppUIBundle). The
	// Server stamps it into the AppContext from the WithUI(...) option AFTER
	// the opt loop is set up; espyna's finalizeHTTPAdapter reads + fail-loud
	// type-asserts every field, and the entydad block reads the auth half.
	// Typed as `any` so this takes no template-FS dependency.
	UI any // *AppUIBundle

	// WorkspaceLoader is the per-request sidebar workspace switcher loader.
	// The entydad block sets it (the proto-backed impl imports workspacepb,
	// illegal in espyna); the Server asserts consumerhttp.WorkspaceLoader.
	WorkspaceLoader any

	// === CSRF injection slots (Wave B D1) ===
	// Set by the Server BEFORE the opt loop via resolveSecurity() so the
	// entydad block reads the SAME resolved secret / cookie-secure / issuer
	// the legacy finalizePreset path uses (single source of truth, one env
	// read, one boot fatal).

	// CSRFSecret is the resolved HMAC secret. Type-assert to []byte.
	CSRFSecret any
	// CookieSecure is the resolved cookie-secure policy. Type-assert to bool.
	CookieSecure any
	// CSRFIssuer issues a fresh workspace-claim CSRF cookie. Type-assert to
	// func(w http.ResponseWriter, secret []byte, sessionToken, workspaceID string) string.
	CSRFIssuer any

	// === Auth-chain finalize slots (Wave B D2a — precondition P1) ===
	// The relocated entydad auth chain (block/buildAuthDeps) reads these off
	// the AppContext instead of off the app's *appBuilder fields. The host
	// stamps them app-side in buildAppContextBase BEFORE the opt loop; the
	// entydad EngineBlock closure type-asserts them when reconstructing the
	// 19-field auth.Deps. Typed as `any` so this takes no espyna-internal dep.

	// SessionManager writes/clears the session cookie on HTTP responses.
	// Type-assert to *consumer.SessionMiddleware (satisfies auth.SessionManager)
	// inside the entydad block. nil when the session middleware is unavailable.
	SessionManager any

	// AuthAdapter performs credential operations (login, register, reset,
	// change-password, validate/invalidate session). Type-assert to
	// *consumer.AuthAdapter inside the entydad block; the block wraps it in
	// its authAdapterBridge to satisfy auth.AuthAdapter. nil when unavailable.
	AuthAdapter any
}

// AppUIBundle is the COMPLETE app-supplied UI/labels contract the espyna
// consumer/http adapter (and the entydad auth.Deps) need from the host (Wave B
// D1, codex round-2). The app builds it once from its own renderer / labels /
// sidebars / translations and passes it via the Server's WithUI(...) option;
// finalizeHTTPAdapter type-asserts each field to its concrete type and
// boot-fatals on a missing or wrong-type slot (fail-closed — a non-mock binary
// must NEVER serve a page with a nil renderer / empty labels / nil translation
// table). Every field is `any`-typed because this package imports no app-
// template-FS package; the concretely-typeable ones are typed in a later cut.
type AppUIBundle struct {
	Renderer         any // *pyeza.HTMLRenderer
	RenderIcon       any // func(string) template.HTML
	CommonLabels     any // pyeza.CommonLabels  (also mirrored into AppContext.Common)
	TableLabels      any // types.TableLabels   (also mirrored into AppContext.Table)
	Messages         any // map[string]string
	Translations     any // *lynguaV1.TranslationProvider (also mirrored into AppContext.Translations)
	SidebarBuilder   any // consumerhttp.SidebarBuilder
	BottomNavBuilder any // consumerhttp.SidebarBuilder
	PortalSidebars   any // map[consumerhttp.PrincipalType]consumerhttp.SidebarBuilder
	ExtLabels        any // the app's extended sidebar/label set
	RouteRewriter    any // func(context.Context) context.Context (calls the app's nav.WithWorkspace)
	AuthLabels       any // auth.AuthLabels (D2: Login02/Signup02/ResetPassword02/ChangePassword/Common/Messages)
}
