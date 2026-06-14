// Package http — workspace_binding.go
//
// Conversions between the consumer-level neutral principal type
// (consumer.PrincipalData, produced/consumed by the ResolveBinding +
// SwitchPrincipal use-case bridges) and the agnostic middleware binding
// (consumermw.WorkspaceBinding, carried across the consumer/http <-> contrib/http
// boundary). Both are framework-agnostic, proto-aligned, scalar-only structs;
// this file is just the field-for-field mapping the WorkspacePath wiring needs.
package http

import (
	"github.com/erniealice/espyna-golang/consumer"
	consumermw "github.com/erniealice/espyna-golang/consumer/http/middleware"
)

// principalDataToBinding maps the use-case bridge's PrincipalData to the
// agnostic WorkspaceBinding the WorkspacePath middleware consumes.
func principalDataToBinding(pd *consumer.PrincipalData) *consumermw.WorkspaceBinding {
	if pd == nil {
		return nil
	}
	b := &consumermw.WorkspaceBinding{
		Kind:        pd.Type,
		PrincipalID: pd.PrincipalID,
		WorkspaceID: pd.WorkspaceID,
		DisplayName: pd.DisplayName,
	}
	if len(pd.ActingAsTargets) > 0 {
		b.ActingAsTargets = make([]consumermw.WorkspaceActingAsTarget, 0, len(pd.ActingAsTargets))
		for _, t := range pd.ActingAsTargets {
			b.ActingAsTargets = append(b.ActingAsTargets, consumermw.WorkspaceActingAsTarget{
				ID:          t.ID,
				WorkspaceID: t.WorkspaceID,
				DisplayName: t.DisplayName,
			})
		}
	}
	return b
}

// bindingToPrincipalData maps the agnostic WorkspaceBinding back to the
// PrincipalData the SwitchPrincipal bridge expects.
func bindingToPrincipalData(b *consumermw.WorkspaceBinding) consumer.PrincipalData {
	if b == nil {
		return consumer.PrincipalData{}
	}
	pd := consumer.PrincipalData{
		Type:        b.Kind,
		PrincipalID: b.PrincipalID,
		WorkspaceID: b.WorkspaceID,
		DisplayName: b.DisplayName,
	}
	if len(b.ActingAsTargets) > 0 {
		pd.ActingAsTargets = make([]consumer.ActingAsTargetData, 0, len(b.ActingAsTargets))
		for _, t := range b.ActingAsTargets {
			pd.ActingAsTargets = append(pd.ActingAsTargets, consumer.ActingAsTargetData{
				ID:          t.ID,
				WorkspaceID: t.WorkspaceID,
				DisplayName: t.DisplayName,
			})
		}
	}
	return pd
}
