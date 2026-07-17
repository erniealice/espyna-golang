package client

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_attribute"
)

// AttributeRepositories groups the three repositories the client create/update
// use cases need to resolve, validate, and persist client_attribute rows atomically
// with the parent client. All three are optional (nil-safe): when unset, submitted
// attributes are rejected fail-closed (a value the server cannot validate is never
// written) rather than silently dropped.
//
// This is the REAL server-side validator mandated by Q-GSE-10 rider #1 (research/D1):
// today's CreateClientAttributeUseCase.validateBusinessRules is a non-empty/≤1000-char
// stub that accepts "banana" under attr-gender. The generic *_attribute EAV overlay
// carries typed constraint columns since W1 (Attribute.{min_value,max_value,
// min_length,max_length,required} + AttributeValue enum rows with a label); this file
// finally reads them.
type AttributeRepositories struct {
	Attribute       commonpb.AttributeDomainServiceServer         // definition + typed constraints (module/data_type/min/max/required)
	AttributeValue  commonpb.AttributeValueDomainServiceServer    // enum option rows (select membership)
	ClientAttribute clientattributepb.ClientAttributeDomainServiceServer // the child value rows we create/sync/delete
}

// enabled reports whether the attribute pipeline is wired (all three repos present).
func (a AttributeRepositories) enabled() bool {
	return a.Attribute != nil && a.AttributeValue != nil && a.ClientAttribute != nil
}

// isApplicableAttributeModule is the SERVER-SIDE mirror of the drawer's
// isClientAttributeModule (entydad .../client/action/action.go). A client can
// only carry attribute definitions whose module is one of the entity-scope
// buckets; a definition from any other module (e.g. "product", "supplier") is
// NOT resolvable on the client path. Without this filter (W3-HIGH-2) a forged
// POST carrying the code of an active product-module definition would be written
// into client_attribute — the drawer filters, the server MUST filter too.
func isApplicableAttributeModule(module string) bool {
	switch module {
	case "entity", "client", "general":
		return true
	default:
		return false
	}
}

// attrControl is the derived form-control family for a definition's data_type.
type attrControl struct {
	control string // "text" | "number" | "select" | "boolean" | "date"
	integer bool   // number sub-kind: true => integral values only (step 1)
}

// classifyAttributeControl maps the definition's free-string data_type to a control
// family, deriving constraints downstream. Repoints the historically-dead
// Attribute.data_type onto the proven ProductOption vocabulary (D1 rider #2) while
// still honouring the education1 live value ("option"→select).
//
// ok=false => unknown data_type: the caller OMITS the field (drawer) or REJECTS the
// submission (validator) — fail-closed, never a silent accept.
func classifyAttributeControl(dataType string) (attrControl, bool) {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "text", "string", "free_text":
		return attrControl{control: "text"}, true
	case "integer", "int", "number_list":
		return attrControl{control: "number", integer: true}, true
	case "number", "decimal", "float", "free_number":
		return attrControl{control: "number", integer: false}, true
	case "option", "enum", "select", "text_list", "color_list":
		return attrControl{control: "select"}, true
	case "boolean", "bool":
		return attrControl{control: "boolean"}, true
	case "date":
		return attrControl{control: "date"}, true
	default:
		return attrControl{}, false
	}
}

// validateAttributeValue enforces the definition's typed constraints against a
// submitted value. A blank value is legal for an optional attribute (it clears the
// row) but rejected when the definition is required. enumSet is the active option
// membership set (values, lowercased-as-stored) — required only for select control.
// Returns the canonical value to persist (booleans normalise to "true"/"false").
func validateAttributeValue(def *commonpb.Attribute, kind attrControl, value string, enumSet map[string]bool) (string, error) {
	code := def.GetCode()
	trimmed := strings.TrimSpace(value)

	// Blank handling — clear vs required.
	if trimmed == "" {
		if def.GetRequired() {
			return "", fmt.Errorf("attribute %q is required", code)
		}
		return "", nil // blank => clear the row (handled by the sync layer)
	}

	switch kind.control {
	case "text":
		// Length rule (W3-MED-7 divergence resolution): the server measures the
		// rune count of the TRIMMED value — the canonical form we persist. The
		// browser's minlength/maxlength count the raw (untrimmed) input, so the
		// server is intentionally at-least-as-strict: trimmed length ≤ raw length,
		// therefore the server can only ever REJECT (never accept) where the
		// browser would — a fail-closed divergence, never a bypass.
		n := utf8.RuneCountInString(trimmed)
		if def.MinLength != nil && n < int(def.GetMinLength()) {
			return "", fmt.Errorf("attribute %q must be at least %d characters", code, def.GetMinLength())
		}
		if def.MaxLength != nil && n > int(def.GetMaxLength()) {
			return "", fmt.Errorf("attribute %q must be at most %d characters", code, def.GetMaxLength())
		}
		return trimmed, nil

	case "number":
		f, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return "", fmt.Errorf("attribute %q must be a number", code)
		}
		// W3-HIGH-3: strconv.ParseFloat accepts "NaN"/"Inf"/"-Inf"; for those both
		// the < min and > max comparisons are false, so a bounds check alone would
		// let a non-finite value through into storage. Reject them explicitly.
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return "", fmt.Errorf("attribute %q must be a finite number", code)
		}
		if kind.integer && f != float64(int64(f)) {
			return "", fmt.Errorf("attribute %q must be a whole number", code)
		}
		if def.MinValue != nil && f < def.GetMinValue() {
			return "", fmt.Errorf("attribute %q must be at least %v", code, def.GetMinValue())
		}
		if def.MaxValue != nil && f > def.GetMaxValue() {
			return "", fmt.Errorf("attribute %q must be at most %v", code, def.GetMaxValue())
		}
		return trimmed, nil

	case "select":
		if !enumSet[trimmed] {
			return "", fmt.Errorf("attribute %q value %q is not an allowed option", code, trimmed)
		}
		return trimmed, nil

	case "boolean":
		// W3-MED-7: accept ONLY the canonical tokens the drawer emits (the boolean
		// select's option values are exactly "true"/"false"). The previous aliases
		// (yes/no/1/0/on/off) were a server-only superset the client never produces
		// — a browser/server contract divergence. Reject anything else.
		switch trimmed {
		case "true":
			return "true", nil
		case "false":
			return "false", nil
		default:
			return "", fmt.Errorf("attribute %q must be a boolean", code)
		}

	case "date":
		if _, err := time.Parse("2006-01-02", trimmed); err != nil {
			return "", fmt.Errorf("attribute %q must be a valid date (YYYY-MM-DD)", code)
		}
		return trimmed, nil

	default:
		return "", fmt.Errorf("attribute %q has an unsupported type", code)
	}
}

// resolvedAttribute is one validated (definition, value) pair ready to persist.
type resolvedAttribute struct {
	attributeID string
	value       string // canonical; "" => clear/delete the client_attribute row
}

// resolveAndValidateAttributes turns the drawer-submitted []AttributeCodeValue into
// validated rows keyed by attribute_id. It resolves each code against the ACTIVE
// definitions in the caller's workspace (the repos are workspace-aware), rejects
// unknown/inactive/ambiguous codes and unknown data_types, and validates every value
// against the typed constraints (rune length, numeric range/integer, enum membership).
// Any error propagates to the caller so the surrounding transaction rolls back.
func resolveAndValidateAttributes(
	ctx context.Context,
	repos AttributeRepositories,
	submitted []*commonpb.AttributeCodeValue,
) ([]resolvedAttribute, error) {
	// NOTE: we do NOT early-return on an empty submission. When the drawer marks
	// the Attributes section present (attributes_present), a REQUIRED applicable
	// definition that is omitted entirely must still be rejected (W3-MED-5) — so
	// the required-coverage sweep below has to run even for zero submitted values.

	// Load active definitions once; build code -> def, rejecting duplicate codes
	// (cross-workspace leakage or a mis-seed) as ambiguous fail-closed. Only
	// definitions whose module is client-applicable ({entity,client,general}) are
	// indexed — W3-HIGH-2: a code from any other module is not resolvable here, so
	// a forged product-module code falls through to the "not an active definition"
	// rejection instead of being written to client_attribute.
	listResp, err := repos.Attribute.ListAttributes(ctx, &commonpb.ListAttributesRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to load attribute definitions: %w", err)
	}
	byCode := make(map[string]*commonpb.Attribute)
	for _, def := range listResp.GetData() {
		if def == nil || !def.GetActive() {
			continue
		}
		if !isApplicableAttributeModule(def.GetModule()) {
			continue
		}
		c := def.GetCode()
		if c == "" {
			continue
		}
		if _, dup := byCode[c]; dup {
			return nil, fmt.Errorf("attribute code %q is ambiguous (multiple active definitions)", c)
		}
		byCode[c] = def
	}

	// enum membership cache keyed by attribute_id.
	enumCache := make(map[string]map[string]bool)
	loadEnum := func(attrID string) (map[string]bool, error) {
		if set, ok := enumCache[attrID]; ok {
			return set, nil
		}
		// Generic list path — preserves the new av.label column (MED#6: the drawer
		// display labels ride the same generic read; the CTE page-queries that drop
		// label are never on this path).
		avResp, err := repos.AttributeValue.ListAttributeValues(ctx, &commonpb.ListAttributeValuesRequest{
			Filters: &commonpb.FilterRequest{
				Filters: []*commonpb.TypedFilter{{
					Field: "attribute_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    attrID,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				}},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to load attribute options: %w", err)
		}
		set := make(map[string]bool)
		for _, av := range avResp.GetData() {
			if av != nil && av.GetActive() {
				set[av.GetValue()] = true
			}
		}
		enumCache[attrID] = set
		return set, nil
	}

	seen := make(map[string]bool)
	resolved := make([]resolvedAttribute, 0, len(submitted))
	for _, sv := range submitted {
		if sv == nil {
			continue
		}
		code := sv.GetCode()
		if code == "" {
			continue
		}
		if seen[code] {
			return nil, fmt.Errorf("attribute code %q submitted more than once", code)
		}
		seen[code] = true

		def, ok := byCode[code]
		if !ok {
			return nil, fmt.Errorf("attribute %q is not an active definition", code)
		}
		kind, known := classifyAttributeControl(def.GetDataType())
		if !known {
			return nil, fmt.Errorf("attribute %q has an unsupported type %q", code, def.GetDataType())
		}
		var enumSet map[string]bool
		if kind.control == "select" {
			enumSet, err = loadEnum(def.GetId())
			if err != nil {
				return nil, err
			}
		}
		canonical, verr := validateAttributeValue(def, kind, sv.GetValue(), enumSet)
		if verr != nil {
			return nil, verr
		}
		resolved = append(resolved, resolvedAttribute{attributeID: def.GetId(), value: canonical})
	}

	// W3-MED-5: required-attribute coverage. When the drawer marks the Attributes
	// section present, EVERY required applicable definition must carry a non-blank
	// submitted value. A required definition submitted blank is already rejected in
	// validateAttributeValue above; a required definition OMITTED ENTIRELY from the
	// submission is caught here — so the browser `required` attribute is backed by a
	// server-side guarantee and cannot be bypassed by a forged/trimmed POST. This
	// sweep runs even for a zero-length submission (the resolve loop above is a
	// no-op then), which is why resolve does NOT early-return on empty input.
	missing := make([]string, 0)
	for code, def := range byCode {
		if def.GetRequired() && !seen[code] {
			missing = append(missing, code)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing) // deterministic error regardless of map order
		return nil, fmt.Errorf("attribute %q is required", missing[0])
	}

	return resolved, nil
}

// syncClientAttributes reconciles the validated rows against the client's existing
// client_attribute rows, IN THE CALLER'S (transaction) context:
//   - blank value  -> delete the existing row (clear), no-op if none exists
//   - changed value -> update the existing row
//   - new value    -> create a new row
// A nil idGen falls back to a timestamp id. Returns the first error (rolls back).
func syncClientAttributes(
	ctx context.Context,
	repos AttributeRepositories,
	idGen ports.IDGenerator,
	clientID string,
	resolved []resolvedAttribute,
) error {
	if clientID == "" {
		return fmt.Errorf("client id is required to sync attributes")
	}

	// Existing rows for this client, keyed by attribute_id.
	existingResp, err := repos.ClientAttribute.ListClientAttributes(ctx, &clientattributepb.ListClientAttributesRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{{
				Field: "client_id",
				FilterType: &commonpb.TypedFilter_StringFilter{
					StringFilter: &commonpb.StringFilter{
						Value:    clientID,
						Operator: commonpb.StringOperator_STRING_EQUALS,
					},
				},
			}},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to load existing client attributes: %w", err)
	}
	existing := make(map[string]*clientattributepb.ClientAttribute)
	for _, row := range existingResp.GetData() {
		if row != nil && row.GetClientId() == clientID {
			existing[row.GetAttributeId()] = row
		}
	}

	newID := func() string {
		if idGen != nil {
			return idGen.GenerateID()
		}
		return fmt.Sprintf("client-attribute-%d", time.Now().UnixNano())
	}

	for _, r := range resolved {
		row, has := existing[r.attributeID]

		if r.value == "" {
			// Clear: delete the existing row (blank optional value never leaves a stale value).
			if has {
				if _, derr := repos.ClientAttribute.DeleteClientAttribute(ctx, &clientattributepb.DeleteClientAttributeRequest{
					Data: &clientattributepb.ClientAttribute{Id: row.GetId()},
				}); derr != nil {
					return fmt.Errorf("failed to clear client attribute: %w", derr)
				}
			}
			continue
		}

		if has {
			if row.GetValue() == r.value && row.GetActive() {
				continue // unchanged
			}
			row.Value = r.value
			row.Active = true
			if _, uerr := repos.ClientAttribute.UpdateClientAttribute(ctx, &clientattributepb.UpdateClientAttributeRequest{
				Data: row,
			}); uerr != nil {
				return fmt.Errorf("failed to update client attribute: %w", uerr)
			}
			continue
		}

		if _, cerr := repos.ClientAttribute.CreateClientAttribute(ctx, &clientattributepb.CreateClientAttributeRequest{
			Data: &clientattributepb.ClientAttribute{
				Id:          newID(),
				ClientId:    clientID,
				AttributeId: r.attributeID,
				Value:       r.value,
				Active:      true,
			},
		}); cerr != nil {
			return fmt.Errorf("failed to create client attribute: %w", cerr)
		}
	}
	return nil
}
