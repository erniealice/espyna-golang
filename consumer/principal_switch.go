// Package consumer — principal_switch.go
//
// Proto-request builder for the SwitchPrincipal service-auth use case.
// All transactional logic (rotation, lock SQL, audit insert, token gen) lives
// in espyna's typed stack — this file just handles the request/response mapping.
//
// Moved from apps/service-admin/internal/composition/ — the proto mapping
// functions and input/result types are framework concerns with no
// app-internal dependencies.

package consumer

import (
	"context"
	"errors"

	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
	principaltypepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/principal_type"
)

// PrincipalSwitchInput is the resolved switch operation. Callers validate
// the target principal before constructing this.
type PrincipalSwitchInput struct {
	UserID             string
	Token              string
	TargetPrincipal    PrincipalData
	ActingAsClientID   string
	ActingAsSupplierID string

	// UseCase tags the rotation in the audit log. Valid values:
	//   switch_url_rotate, switch_url_acting_as_inplace, switch_url_principal_inplace,
	//   switch_explicit_rotate, switch_explicit_inplace, switch_explicit_acting_as.
	// Empty/unrecognized maps to SWITCH_USE_CASE_UNSPECIFIED; the adapter
	// derives the discriminator from URLDriven + rotation/in-place deltas.
	UseCase string

	// Forensic metadata for the audit row.
	RequestURL   string
	Referer      string
	SecFetchSite string
	UserAgent    string

	// URLDriven distinguishes URL-driven (workspace_path middleware) from
	// explicit-form (/action/auth/switch-principal) callers.
	URLDriven bool

	// RequireAudit, when true, causes audit-row insert failure to roll back
	// the rotation transaction (closes red-team A-4).
	RequireAudit bool
}

// PrincipalData holds the target principal fields needed for the switch
// proto request. This is a framework-level type that avoids importing the
// app's internal adapthttp.Principal type.
type PrincipalData struct {
	Type            int32
	PrincipalID     string
	WorkspaceID     string
	DisplayName     string
	ActingAsTargets []ActingAsTargetData
}

// ActingAsTargetData holds delegate acting-as target data for the principal
// switch request.
type ActingAsTargetData struct {
	ID          string
	WorkspaceID string
	DisplayName string
}

// PrincipalSwitchResult tells the HTTP handler what to do with the response.
type PrincipalSwitchResult struct {
	// NewToken is non-empty when rotation occurred (handler must SetSessionCookie).
	NewToken string
	// RedirectURL is where the handler should redirect.
	RedirectURL string
}

// SwitchPrincipalExecutor is the interface for the SwitchPrincipal use case.
type SwitchPrincipalExecutor interface {
	Execute(ctx context.Context, req *authpb.SwitchPrincipalRequest) (*authpb.SwitchPrincipalResponse, error)
}

// ExecutePrincipalSwitch builds a proto request from PrincipalSwitchInput,
// calls the SwitchPrincipal use case, and maps the response back to
// PrincipalSwitchResult.
func ExecutePrincipalSwitch(
	ctx context.Context,
	uc SwitchPrincipalExecutor,
	in PrincipalSwitchInput,
) (*PrincipalSwitchResult, error) {
	if uc == nil {
		return nil, errors.New("principal switch: SwitchPrincipal use case not wired")
	}

	req := &authpb.SwitchPrincipalRequest{
		UserId:             in.UserID,
		Token:              in.Token,
		TargetPrincipal:    ToProtoPrincipal(in.TargetPrincipal),
		ActingAsClientId:   in.ActingAsClientID,
		ActingAsSupplierId: in.ActingAsSupplierID,
		UseCase:            SwitchUseCaseFromString(in.UseCase),
		RequestUrl:         in.RequestURL,
		Referer:            in.Referer,
		SecFetchSite:       in.SecFetchSite,
		UserAgent:          in.UserAgent,
		UrlDriven:          in.URLDriven,
		RequireAudit:       in.RequireAudit,
	}

	resp, err := uc.Execute(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("principal switch: nil response from use case")
	}

	return &PrincipalSwitchResult{
		NewToken:    resp.GetNewToken(),
		RedirectURL: resp.GetRedirectUrl(),
	}, nil
}

// ToProtoPrincipal maps PrincipalData to the proto Principal message.
func ToProtoPrincipal(p PrincipalData) *authpb.Principal {
	out := &authpb.Principal{
		Type:        principaltypepb.PrincipalType(p.Type),
		PrincipalId: p.PrincipalID,
		WorkspaceId: p.WorkspaceID,
		DisplayName: p.DisplayName,
	}
	if len(p.ActingAsTargets) > 0 {
		out.ActingAsTargets = make([]*authpb.ActingAsTarget, 0, len(p.ActingAsTargets))
		for _, t := range p.ActingAsTargets {
			out.ActingAsTargets = append(out.ActingAsTargets, &authpb.ActingAsTarget{
				Id:          t.ID,
				WorkspaceId: t.WorkspaceID,
				DisplayName: t.DisplayName,
			})
		}
	}
	return out
}

// SwitchUseCaseFromString maps a string discriminator to the proto
// SwitchUseCase enum.
func SwitchUseCaseFromString(s string) authpb.SwitchUseCase {
	switch s {
	case "switch_url_rotate":
		return authpb.SwitchUseCase_SWITCH_USE_CASE_URL_ROTATE
	case "switch_url_acting_as_inplace":
		return authpb.SwitchUseCase_SWITCH_USE_CASE_URL_ACTING_AS_INPLACE
	case "switch_url_principal_inplace":
		return authpb.SwitchUseCase_SWITCH_USE_CASE_URL_PRINCIPAL_INPLACE
	case "switch_explicit_rotate":
		return authpb.SwitchUseCase_SWITCH_USE_CASE_EXPLICIT_ROTATE
	case "switch_explicit_inplace":
		return authpb.SwitchUseCase_SWITCH_USE_CASE_EXPLICIT_INPLACE
	case "switch_explicit_acting_as":
		return authpb.SwitchUseCase_SWITCH_USE_CASE_EXPLICIT_ACTING_AS
	}
	return authpb.SwitchUseCase_SWITCH_USE_CASE_UNSPECIFIED
}
