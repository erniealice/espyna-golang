//go:build sqlserver

package audit

// ExcludedFields contains field names that must never appear in audit trail records.
// This is the V1 hardcoded list — will be replaced by per-workspace AuditPolicy in V2.
//
// Fields are excluded because they contain secrets or sensitive authentication data
// that should never be persisted in the audit trail.
// Identical to the postgres gold standard; no dialect differences here.
var ExcludedFields = map[string]bool{
	"password_hash":          true,
	"password_reset_token":   true,
	"password_reset_expires": true,
	"api_key":                true,
	"api_secret":             true,
	"access_token":           true,
	"refresh_token":          true,
	"session_token":          true,
	"secret":                 true,
	"private_key":            true,
	"credentials":            true,
}

// IsExcluded returns true if the field should be skipped during audit logging.
func IsExcluded(fieldName string) bool {
	return ExcludedFields[fieldName]
}
