package subscription

import (
	"strings"

	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

// DefaultCodeFormat is the code template used when SUBSCRIPTION_CODE_FORMAT is
// unset or set to the sentinel "auto". Names come from the STUDENT (the client's
// own first_name/last_name), never from client.user.
const DefaultCodeFormat = "{last_name}, {first_name} ({price_schedule})"

// autoCodeFormatSentinel is the .env value that maps to DefaultCodeFormat.
const autoCodeFormatSentinel = "auto"

// resolveCodeFormat normalizes a configured format string: an empty value or the
// "auto" sentinel (case-insensitive) resolves to DefaultCodeFormat. Any other
// value is returned verbatim so operators can fully control the template.
func resolveCodeFormat(configured string) string {
	trimmed := strings.TrimSpace(configured)
	if trimmed == "" || strings.EqualFold(trimmed, autoCodeFormatSentinel) {
		return DefaultCodeFormat
	}
	return trimmed
}

// FormatCode substitutes {token} placeholders in template with values from
// tokens. It is pure and dependency-free. Unknown tokens (placeholders with no
// matching key) resolve to the empty string — the placeholder is removed, never
// walked as a path. A blank/"auto" template resolves to DefaultCodeFormat.
func FormatCode(template string, tokens map[string]string) string {
	tmpl := resolveCodeFormat(template)

	var b strings.Builder
	b.Grow(len(tmpl))

	for i := 0; i < len(tmpl); {
		if tmpl[i] == '{' {
			if close := strings.IndexByte(tmpl[i+1:], '}'); close >= 0 {
				name := tmpl[i+1 : i+1+close]
				// Known token -> value; unknown token -> "" (removed).
				b.WriteString(tokens[name])
				i = i + 1 + close + 1
				continue
			}
		}
		b.WriteByte(tmpl[i])
		i++
	}

	return b.String()
}

// ResolveCodeTokens builds the fixed, documented token set from the resolved
// student (client), plan, and price schedule. Any nil input contributes empty
// tokens (never an error) so code generation is always best-effort.
//
// Fixed token set (no arbitrary path reflection):
//
//	{first_name}     -> client.first_name
//	{last_name}      -> client.last_name
//	{client_name}    -> client.name, else "first last"
//	{grade}          -> plan.name
//	{price_schedule} -> price_schedule.name
func ResolveCodeTokens(client *clientpb.Client, plan *planpb.Plan, schedule *priceschedulepb.PriceSchedule) map[string]string {
	firstName := ""
	lastName := ""
	clientName := ""
	if client != nil {
		firstName = client.GetFirstName()
		lastName = client.GetLastName()
		clientName = client.GetName()
	}
	if clientName == "" {
		clientName = strings.TrimSpace(strings.TrimSpace(firstName + " " + lastName))
	}

	grade := ""
	if plan != nil {
		grade = plan.GetName()
	}

	priceSchedule := ""
	if schedule != nil {
		priceSchedule = schedule.GetName()
	}

	return map[string]string{
		"first_name":     firstName,
		"last_name":      lastName,
		"client_name":    clientName,
		"grade":          grade,
		"price_schedule": priceSchedule,
	}
}
