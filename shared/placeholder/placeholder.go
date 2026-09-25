// Package placeholder is the single, generic allowlist + renderer for
// placeholder tags embedded in authored text (e.g. rating-description set
// entries). A tag is `{segment(.segment)+}` with segments `[a-z][a-z0-9_]*`
// — e.g. `{client.user.first_name}`. Braces that do not match that grammar are
// literal text and are never touched.
//
// The Registry is the ONLY allowlist: adding a tag = one registry entry here +
// one field→column mapping in the batched loader of the adapter that supplies
// its value (the adapter's own test asserts every allowlisted tag is mapped).
//
// Rendering is single pass: substituted values are never re-scanned, so a
// value that itself contains `{client.user.first_name}` stays inert text.
// Rendering fails CLOSED — an allowlisted tag with an empty/missing value, or
// any unknown grammar-matching tag, is an error (callers never write a
// half-rendered text).
//
// Charter: standard library only. No proto types, no DB drivers, no vertical
// nouns (tags name canonical entities/fields).
package placeholder

import (
	"errors"
	"regexp"
	"sort"
	"strings"
)

// Error codes (bounded, non-value-echoing — safe for acks and logs).
const (
	// CodeUnknownPlaceholder: the text contains a grammar-matching tag that is
	// not in the allowlist.
	CodeUnknownPlaceholder = "UNKNOWN_PLACEHOLDER"
	// CodeMissingValue: an allowlisted tag has no (or a blank) value for the
	// record being rendered.
	CodeMissingValue = "PLACEHOLDER_VALUE_MISSING"
)

// Tag names (canonical entity.field paths — no vertical vocabulary).
const (
	// TagClientUserFirstName is the client's linked user's first name.
	TagClientUserFirstName = "client.user.first_name"
)

// Sentinels for errors.Is matching against *Error.
var (
	ErrUnknownPlaceholder = errors.New(CodeUnknownPlaceholder)
	ErrMissingValue       = errors.New(CodeMissingValue)
)

// Error is the typed placeholder failure. Code is one of the Code* constants;
// Tags lists the offending tags (distinct, first-appearance order).
type Error struct {
	Code string
	Tags []string
}

func (e *Error) Error() string {
	return "placeholder: " + e.Code + ": " + strings.Join(e.Tags, ", ")
}

// Is lets errors.Is(err, ErrUnknownPlaceholder / ErrMissingValue) match.
func (e *Error) Is(target error) bool {
	switch target {
	case ErrUnknownPlaceholder:
		return e.Code == CodeUnknownPlaceholder
	case ErrMissingValue:
		return e.Code == CodeMissingValue
	}
	return false
}

// tagPattern is the tag grammar; group 1 is the dotted tag path.
var tagPattern = regexp.MustCompile(`\{([a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+)\}`)

// registry is the allowlist. Keep it tiny and explicit.
var registry = map[string]struct{}{
	TagClientUserFirstName: {},
}

// Allowed returns the allowlisted tags, sorted.
func Allowed() []string {
	out := make([]string, 0, len(registry))
	for t := range registry {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// IsAllowed reports whether tag (without braces) is allowlisted.
func IsAllowed(tag string) bool {
	_, ok := registry[tag]
	return ok
}

// Root returns the first segment of a tag ("client" for
// "client.user.first_name") — the entity whose loader supplies its value.
func Root(tag string) string {
	if i := strings.IndexByte(tag, '.'); i >= 0 {
		return tag[:i]
	}
	return tag
}

// Tags returns every grammar-matching tag in text (allowlisted or not),
// distinct, in first-appearance order, without braces.
func Tags(text string) []string {
	matches := tagPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(matches))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	return out
}

// ValidateAllowed returns an *Error with CodeUnknownPlaceholder listing every
// grammar-matching tag in text that is not allowlisted; nil otherwise.
func ValidateAllowed(text string) error {
	var unknown []string
	for _, t := range Tags(text) {
		if !IsAllowed(t) {
			unknown = append(unknown, t)
		}
	}
	if len(unknown) > 0 {
		return &Error{Code: CodeUnknownPlaceholder, Tags: unknown}
	}
	return nil
}

// Render substitutes every tag in text with values[tag] in a single pass
// (substituted values are never re-scanned). Text without tags is returned
// unchanged. Fails closed:
//   - any non-allowlisted tag  → *Error{CodeUnknownPlaceholder}
//   - any allowlisted tag whose value is missing or blank (whitespace only)
//     → *Error{CodeMissingValue} listing every such tag.
//
// On error the returned string is "" — never a half-rendered text.
func Render(text string, values map[string]string) (string, error) {
	tags := Tags(text)
	if len(tags) == 0 {
		return text, nil
	}
	if err := ValidateAllowed(text); err != nil {
		return "", err
	}
	var missing []string
	for _, t := range tags {
		if strings.TrimSpace(values[t]) == "" {
			missing = append(missing, t)
		}
	}
	if len(missing) > 0 {
		return "", &Error{Code: CodeMissingValue, Tags: missing}
	}
	return tagPattern.ReplaceAllStringFunc(text, func(m string) string {
		// m is the full `{tag}` match; strip the braces.
		return values[m[1:len(m)-1]]
	}), nil
}
