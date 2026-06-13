package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	sessionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/session"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

// defaultSessionExpiry is the fallback TTL when Services.Expiry is zero.
const defaultSessionExpiry = 7 * 24 * time.Hour

// sessionTokenBytes is the number of random bytes in a session token before
// hex encoding. 32 bytes = 256 bits of entropy, matching the legacy
// password.SessionService contract.
const sessionTokenBytes = 32

// IssueSessionRepositories is the write-side of session lifecycle.
type IssueSessionRepositories struct {
	Session sessionpb.SessionDomainServiceServer
}

// IssueSessionServices groups infrastructure deps. No Authorizer —
// this use case establishes identity.
type IssueSessionServices struct {
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
	Expiry      SessionExpiryConfig
}

// IssueSessionUseCase mints a cryptographically random token and persists the
// backing session row.
type IssueSessionUseCase struct {
	repositories IssueSessionRepositories
	services     IssueSessionServices
}

// NewIssueSessionUseCase wires the use case.
func NewIssueSessionUseCase(
	repositories IssueSessionRepositories,
	services IssueSessionServices,
) *IssueSessionUseCase {
	return &IssueSessionUseCase{repositories: repositories, services: services}
}

// Execute creates a session row and returns the opaque bearer token as a
// proto IssueSessionResponse.
func (uc *IssueSessionUseCase) Execute(
	ctx context.Context,
	req *authpb.IssueSessionRequest,
) (*authpb.IssueSessionResponse, error) {
	if uc.repositories.Session == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.errors.service_unavailable", "Auth service is not available [DEFAULT]"))
	}
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *IssueSessionUseCase) executeWithTransaction(
	ctx context.Context,
	req *authpb.IssueSessionRequest,
) (*authpb.IssueSessionResponse, error) {
	var out *authpb.IssueSessionResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translated := contextutil.GetTranslatedMessageWithContext(
				txCtx, uc.services.Translator,
				"auth.errors.issue_session_failed", "Failed to issue session [DEFAULT]")
			return fmt.Errorf("%s: %w", translated, err)
		}
		out = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (uc *IssueSessionUseCase) executeCore(
	ctx context.Context,
	req *authpb.IssueSessionRequest,
) (*authpb.IssueSessionResponse, error) {
	token, err := generateSessionToken()
	if err != nil {
		return nil, err
	}

	sessionID := ""
	if uc.services.IDGenerator != nil {
		sessionID = uc.services.IDGenerator.GenerateID()
	}

	ttl := defaultSessionExpiry
	if uc.services.Expiry.Duration > 0 {
		ttl = uc.services.Expiry.Duration
	}
	expiresAt := time.Now().Add(ttl).UnixMilli()

	data := &sessionpb.Session{
		Id:        sessionID,
		UserId:    req.GetUserId(),
		Token:     token,
		ExpiresAt: expiresAt,
		Active:    true,
	}
	if wsu := req.GetWorkspaceUserId(); wsu != "" {
		wsuCopy := wsu
		data.WorkspaceUserId = &wsuCopy
	}
	if ws := req.GetWorkspaceId(); ws != "" {
		wsCopy := ws
		data.WorkspaceId = &wsCopy
	}

	resp, err := uc.repositories.Session.CreateSession(ctx, &sessionpb.CreateSessionRequest{Data: data})
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.errors.issue_session_failed", "Failed to issue session [DEFAULT]"))
	}
	created := resp.Data[0]

	out := &authpb.IssueSessionResponse{
		Token:           created.Token,
		SessionId:       created.Id,
		ExpiresAtUnixMs: created.ExpiresAt,
	}
	if created.WorkspaceUserId != nil {
		out.WorkspaceUserId = *created.WorkspaceUserId
	}
	if created.WorkspaceId != nil {
		out.WorkspaceId = *created.WorkspaceId
	}
	return out, nil
}

func (uc *IssueSessionUseCase) validateInput(ctx context.Context, req *authpb.IssueSessionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.validation.request_required", "Session issuance request is required [DEFAULT]"))
	}
	if req.GetUserId() == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.validation.user_required", "User ID is required to issue a session [DEFAULT]"))
	}
	return nil
}

// generateSessionToken returns a cryptographically random, hex-encoded token.
func generateSessionToken() (string, error) {
	buf := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
