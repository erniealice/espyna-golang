// Package consumer — auth_aliases.go.
//
// DEPRECATED — this file is a temporary visibility bridge introduced in
// 20260518-hexagonal-strict-adherence Phase 1.D. It exists only because
// Go's internal/ rule blocks apps from importing
// internal/application/usecases/auth directly, while service-admin's
// session middleware needs to construct AuthenticateSession / IssueSession /
// InvalidateSession Request types.
//
// SCHEDULED FOR REMOVAL by the service-domain-migration follow-up plan
// (docs/plan/20260520-service-domain-migration/) — which migrates auth to
// the service-driven proto category proto/v1/service/auth/. Once that plan
// ships, apps construct proto-shaped Request types directly and this file
// is deleted. Do not add new aliases here.
//
// TODO: delete this file together with the auth/session migration.
package consumer

import (
	"github.com/erniealice/espyna-golang/internal/application/usecases/auth"
)

// Auth use case Request/Response shapes (re-exported as type aliases).
type AuthenticateSessionRequest = auth.AuthenticateSessionRequest
type AuthenticateSessionResponse = auth.AuthenticateSessionResponse
type AuthIdentity = auth.Identity
type IssueSessionRequest = auth.IssueSessionRequest
type IssueSessionResponse = auth.IssueSessionResponse
type InvalidateSessionRequest = auth.InvalidateSessionRequest
