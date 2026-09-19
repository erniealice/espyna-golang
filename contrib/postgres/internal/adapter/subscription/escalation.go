//go:build postgresql

package subscription

import (
	"database/sql"

	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

func escalationModeToken(mode priceplanpb.EscalationMode) any {
	switch mode {
	case priceplanpb.EscalationMode_ESCALATION_MODE_NONE:
		return "none"
	case priceplanpb.EscalationMode_ESCALATION_MODE_FIXED_PERCENTAGE:
		return "fixed_percentage"
	default:
		return nil
	}
}

func escalationScopeToken(scope *priceplanpb.EscalationScope) any {
	if scope == nil {
		return nil
	}
	switch *scope {
	case priceplanpb.EscalationScope_ESCALATION_SCOPE_WITHIN_AGREEMENT:
		return "within_agreement"
	case priceplanpb.EscalationScope_ESCALATION_SCOPE_ON_RENEWAL:
		return "on_renewal"
	default:
		return nil
	}
}

func optionalInt32Value(value *int32) any {
	if value == nil {
		return nil
	}
	return *value
}

func writeEscalationTuple(data map[string]any, prefix string, mode *priceplanpb.EscalationMode, scope *priceplanpb.EscalationScope, rate, first, every *int32) {
	if mode == nil {
		return
	}
	base := prefix + "Escalation"
	if prefix == "" {
		base = "escalation"
	}
	data[base+"Mode"] = escalationModeToken(*mode)
	data[base+"Scope"] = escalationScopeToken(scope)
	data[base+"RateBps"] = optionalInt32Value(rate)
	data[base+"FirstAfterMonths"] = optionalInt32Value(first)
	data[base+"EveryMonths"] = optionalInt32Value(every)
}

func normalizeEscalationReadMap(data map[string]any, prefix string) {
	base := prefix + "Escalation"
	if prefix == "" {
		base = "escalation"
	}
	modeKeys := []string{base + "Mode"}
	scopeKeys := []string{base + "Scope"}
	if prefix == "default" {
		modeKeys = append(modeKeys, "default_escalation_mode")
		scopeKeys = append(scopeKeys, "default_escalation_scope")
	} else {
		modeKeys = append(modeKeys, "escalation_mode")
		scopeKeys = append(scopeKeys, "escalation_scope")
	}
	for _, key := range modeKeys {
		switch data[key] {
		case "none":
			data[key] = "ESCALATION_MODE_NONE"
		case "fixed_percentage":
			data[key] = "ESCALATION_MODE_FIXED_PERCENTAGE"
		}
	}
	for _, key := range scopeKeys {
		switch data[key] {
		case "within_agreement":
			data[key] = "ESCALATION_SCOPE_WITHIN_AGREEMENT"
		case "on_renewal":
			data[key] = "ESCALATION_SCOPE_ON_RENEWAL"
		}
	}
}

func escalationModeFromToken(token string) (priceplanpb.EscalationMode, bool) {
	switch token {
	case "none":
		return priceplanpb.EscalationMode_ESCALATION_MODE_NONE, true
	case "fixed_percentage":
		return priceplanpb.EscalationMode_ESCALATION_MODE_FIXED_PERCENTAGE, true
	default:
		return priceplanpb.EscalationMode_ESCALATION_MODE_UNSPECIFIED, false
	}
}

func escalationScopeFromToken(token string) (priceplanpb.EscalationScope, bool) {
	switch token {
	case "within_agreement":
		return priceplanpb.EscalationScope_ESCALATION_SCOPE_WITHIN_AGREEMENT, true
	case "on_renewal":
		return priceplanpb.EscalationScope_ESCALATION_SCOPE_ON_RENEWAL, true
	default:
		return priceplanpb.EscalationScope_ESCALATION_SCOPE_UNSPECIFIED, false
	}
}

func applyPricePlanEscalation(pp *priceplanpb.PricePlan, mode, scope sql.NullString, rate, first, every sql.NullInt32) {
	if value, ok := escalationModeFromToken(mode.String); mode.Valid && ok {
		pp.DefaultEscalationMode = &value
	}
	if value, ok := escalationScopeFromToken(scope.String); scope.Valid && ok {
		pp.DefaultEscalationScope = &value
	}
	if rate.Valid {
		value := rate.Int32
		pp.DefaultEscalationRateBps = &value
	}
	if first.Valid {
		value := first.Int32
		pp.DefaultEscalationFirstAfterMonths = &value
	}
	if every.Valid {
		value := every.Int32
		pp.DefaultEscalationEveryMonths = &value
	}
}

func applySubscriptionEscalation(s *subscriptionpb.Subscription, mode, scope sql.NullString, rate, first, every sql.NullInt32) {
	if value, ok := escalationModeFromToken(mode.String); mode.Valid && ok {
		s.EscalationMode = &value
	}
	if value, ok := escalationScopeFromToken(scope.String); scope.Valid && ok {
		s.EscalationScope = &value
	}
	if rate.Valid {
		value := rate.Int32
		s.EscalationRateBps = &value
	}
	if first.Valid {
		value := first.Int32
		s.EscalationFirstAfterMonths = &value
	}
	if every.Valid {
		value := every.Int32
		s.EscalationEveryMonths = &value
	}
}
