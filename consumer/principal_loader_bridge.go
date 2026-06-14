// Package consumer -- principal_loader_bridge.go
//
// Proto-to-local principal type mapping utilities for the principal resolver
// bridge. These are pure functions that convert between the proto Principal
// types (esqyma) and a local PrincipalData representation.
//
// Moved from apps/service-admin/internal/composition/.

package consumer

import (
	"context"
	"errors"

	principaltypepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/principal_type"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

// ResolvePrincipalsExecutor is the interface for the ResolvePrincipals use case.
type ResolvePrincipalsExecutor interface {
	Execute(ctx context.Context, req *authpb.ResolvePrincipalsRequest) (*authpb.ResolvePrincipalsResponse, error)
}

// ResolveBindingExecutor is the interface for the ResolveBinding use case.
type ResolveBindingExecutor interface {
	Execute(ctx context.Context, userID, workspaceID string, sessionPrincipalKind principaltypepb.PrincipalType, sessionPrincipalID string) (*authpb.Principal, error)
}

// PrincipalResolveFunc is the function signature for resolving all principals
// for a user, returning framework-level PrincipalData values.
type PrincipalResolveFunc func(ctx context.Context, userID string) ([]PrincipalData, error)

// BindingResolveFunc is the function signature for resolving a single binding
// in one workspace, returning a framework-level PrincipalData value.
type BindingResolveFunc func(ctx context.Context, userID, workspaceID string, sessionPrincipalKind int32, sessionPrincipalID string) (*PrincipalData, error)

// BuildPrincipalResolveFn constructs a PrincipalResolveFunc from a
// ResolvePrincipals use case executor.
func BuildPrincipalResolveFn(uc ResolvePrincipalsExecutor) PrincipalResolveFunc {
	if uc == nil {
		return nil
	}
	return func(ctx context.Context, userID string) ([]PrincipalData, error) {
		resp, err := uc.Execute(ctx, &authpb.ResolvePrincipalsRequest{
			UserId: userID,
		})
		if err != nil {
			return nil, err
		}
		return ProtoPrincipalsToData(resp.GetPrincipals()), nil
	}
}

// BuildBindingResolveFn constructs a BindingResolveFunc from a ResolveBinding
// use case executor. Returns nil if the executor is nil.
func BuildBindingResolveFn(uc ResolveBindingExecutor) BindingResolveFunc {
	if uc == nil {
		return nil
	}
	return func(ctx context.Context, userID, workspaceID string, sessionPrincipalKind int32, sessionPrincipalID string) (*PrincipalData, error) {
		binding, err := uc.Execute(ctx,
			userID, workspaceID,
			principaltypepb.PrincipalType(sessionPrincipalKind),
			sessionPrincipalID,
		)
		if err != nil {
			return nil, err
		}
		if binding == nil {
			return nil, errors.New("resolve_binding: no active binding in workspace")
		}
		p := ProtoPrincipalToData(binding)
		return &p, nil
	}
}

// ProtoPrincipalsToData converts a slice of proto Principal messages to
// framework-level PrincipalData values.
func ProtoPrincipalsToData(pbs []*authpb.Principal) []PrincipalData {
	out := make([]PrincipalData, 0, len(pbs))
	for _, pb := range pbs {
		out = append(out, ProtoPrincipalToData(pb))
	}
	return out
}

// ProtoPrincipalToData converts a single proto Principal message to a
// framework-level PrincipalData value.
func ProtoPrincipalToData(pb *authpb.Principal) PrincipalData {
	if pb == nil {
		return PrincipalData{}
	}
	targets := make([]ActingAsTargetData, 0, len(pb.ActingAsTargets))
	for _, t := range pb.ActingAsTargets {
		targets = append(targets, ActingAsTargetData{
			ID:          t.Id,
			WorkspaceID: t.WorkspaceId,
			DisplayName: t.DisplayName,
		})
	}
	return PrincipalData{
		Type:            int32(pb.Type),
		PrincipalID:     pb.PrincipalId,
		WorkspaceID:     pb.WorkspaceId,
		DisplayName:     pb.DisplayName,
		ActingAsTargets: targets,
	}
}
