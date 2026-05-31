package attachment

import (
	"sort"
	"strings"
)

// Q-ST-POLICY (LOCKED, B->C): the central default-deny upload registry shipped in
// hybra (9c81aee, views/attachment/policy.go) as the HTTP-surface gate. W4 pushes
// that policy DOWN into this espyna use-case package so CreateAttachment enforces it
// server-side for EVERY caller — not just the hybra upload handler. The hybra
// handler may still short-circuit early (cheaper UX rejection), but this use-case
// assertion is the authoritative backstop.
//
// This file is the espyna-side port of the registry. It is intentionally a small,
// self-contained copy (espyna cannot import hybra — hybra depends on espyna, not the
// reverse) keyed by module_key. DEFAULT-DENY semantics are preserved: a module_key
// with no entry yields a zero Policy, which rejects every content type.

// Policy is the per-module (module_key) upload allow-list enforced at the
// CreateAttachment use case.
type Policy struct {
	// AllowedContentTypes is the set of accepted MIME types for this module. The
	// comparison is case-insensitive and ignores any "; charset=" parameter. An
	// empty set denies everything (default-deny).
	//
	// NOTE: at the use-case layer we no longer hold the raw bytes, so we cannot
	// re-run http.DetectContentType here. We instead validate the SERVER-PERSISTED
	// content_type carried on the attachment row (req.Data.ContentType). The hybra
	// upload handler is responsible for deriving that value from a magic-byte sniff
	// (never the raw client header) BEFORE calling CreateAttachment (ST-H3); this
	// use case then asserts the persisted value is on the module's allow-list. A
	// non-hybra caller that fabricates a content_type still cannot smuggle a
	// disallowed type past this gate, because the allow-list itself is the ceiling.
	AllowedContentTypes []string

	// MaxFileCount is the maximum number of ACTIVE attachments allowed on a single
	// parent record (module_key, foreign_key). 0 means "no count cap enforced".
	MaxFileCount int
}

// allowsContentType reports whether the (normalized) content type is on the
// module's allow-list. A policy with no allowed types denies everything.
func (p Policy) allowsContentType(contentType string) bool {
	if len(p.AllowedContentTypes) == 0 {
		return false
	}
	base := normalizeContentType(contentType)
	if base == "" {
		return false
	}
	for _, ct := range p.AllowedContentTypes {
		if normalizeContentType(ct) == base {
			return true
		}
	}
	return false
}

// normalizeContentType lower-cases a MIME type and strips any parameters
// (e.g. "text/plain; charset=utf-8" -> "text/plain").
func normalizeContentType(ct string) string {
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	return strings.ToLower(strings.TrimSpace(ct))
}

// ----------------------------------------------------------------------------
// Reusable policy building blocks (mirrors hybra commonSafe / imagesOnly)
// ----------------------------------------------------------------------------

// commonSafeContentTypes is the conservative "documents + images" allow-list, as
// seen by http.DetectContentType (Office OOXML containers sniff as application/zip,
// legacy OLE docs as application/x-ole-storage, text/csv as text/plain, etc.).
var commonSafeContentTypes = []string{
	"application/pdf",
	"image/png",
	"image/jpeg",
	"image/gif",
	"image/webp",
	"image/bmp",
	"image/tiff",
	"text/plain",
	"text/csv",
	"application/zip",              // .docx/.xlsx/.pptx (OOXML are zip)
	"application/x-zip-compressed", // some platforms label zip this way
	"application/octet-stream",     // some OOXML / older formats sniff generic
	"application/x-ole-storage",    // legacy .doc/.xls (OLE compound file)
}

// imagesOnlyContentTypes is the tighter image-only surface (catalog photos).
var imagesOnlyContentTypes = []string{
	"image/png", "image/jpeg", "image/gif", "image/webp", "image/bmp", "image/tiff",
}

// commonSafePolicy is documents + images with a 20-file per-record cap (W4: the
// hybra builders left MaxFileCount at 0 with a "cap per module in W4" TODO; this
// is the espyna-authoritative value).
func commonSafePolicy() Policy {
	return Policy{
		AllowedContentTypes: commonSafeContentTypes,
		MaxFileCount:        20,
	}
}

// imagesOnlyPolicy restricts a surface to images with a 10-file per-record cap.
func imagesOnlyPolicy() Policy {
	return Policy{
		AllowedContentTypes: imagesOnlyContentTypes,
		MaxFileCount:        10,
	}
}

// ----------------------------------------------------------------------------
// Central per-module registry (keyed by module_key)
// ----------------------------------------------------------------------------

// defaultPolicyRegistry maps a module_key to its upload Policy. It mirrors the
// hybra DefaultRegistry seed so the two layers agree. DEFAULT-DENY: a module_key
// with no entry resolves to a zero Policy (rejects every upload).
var defaultPolicyRegistry = map[string]Policy{
	// --- centymo: commerce records (documents + images) ---
	"accrued_expense":                  commonSafePolicy(),
	"collection":                       commonSafePolicy(),
	"disbursement":                     commonSafePolicy(),
	"expenditure":                      commonSafePolicy(),
	"expense_recognition":              commonSafePolicy(),
	"inventory":                        commonSafePolicy(),
	"plan":                             commonSafePolicy(),
	"price_plan":                       commonSafePolicy(),
	"price_schedule":                   commonSafePolicy(),
	"pricelist":                        commonSafePolicy(),
	"procurement_request":              commonSafePolicy(),
	"purchase_order":                   commonSafePolicy(),
	"revenue":                          commonSafePolicy(),
	"revenue_run":                      commonSafePolicy(),
	"subscription":                     commonSafePolicy(),
	"supplier_contract":                commonSafePolicy(),
	"supplier_contract_price_schedule": commonSafePolicy(),
	"line":                             commonSafePolicy(),

	// product surfaces carry catalog photos -> images-only.
	"product":    imagesOnlyPolicy(),
	"variant":    imagesOnlyPolicy(),
	"stock-item": imagesOnlyPolicy(),

	// --- entydad: identity / org records (documents + images) ---
	"client":         commonSafePolicy(),
	"supplier":       commonSafePolicy(),
	"user":           commonSafePolicy(),
	"workspace":      commonSafePolicy(),
	"workspace_user": commonSafePolicy(),
	"location":       commonSafePolicy(),
	"role":           commonSafePolicy(),

	// --- fayna: operations / jobs (documents + images) ---
	"fulfillment":      commonSafePolicy(),
	"job":              commonSafePolicy(),
	"job_activity":     commonSafePolicy(),
	"job_phase":        commonSafePolicy(),
	"job_task":         commonSafePolicy(),
	"job_template":     commonSafePolicy(),
	"outcome_criteria": commonSafePolicy(),
	"task_outcome":     commonSafePolicy(),

	// --- fycha: accounting / assets (documents + images) ---
	"asset":         commonSafePolicy(),
	"journal_entry": commonSafePolicy(),

	// --- cyta: scheduling (documents + images) ---
	"event": commonSafePolicy(),
}

// policyFor returns the effective Policy for a module_key. A module_key with no
// registered policy resolves to a zero Policy (DEFAULT-DENY: rejects every upload).
func policyFor(moduleKey string) Policy {
	if p, ok := defaultPolicyRegistry[moduleKey]; ok {
		return p
	}
	return Policy{}
}

// registeredPolicyModuleKeys returns the sorted set of module keys with an explicit
// policy (boot-time auditing / tests).
func registeredPolicyModuleKeys() []string {
	keys := make([]string, 0, len(defaultPolicyRegistry))
	for k := range defaultPolicyRegistry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
