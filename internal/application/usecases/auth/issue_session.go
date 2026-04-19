package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	sessionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/session"
)

// defaultSessionExpiry is the fallback TTL when Services.SessionExpiry is zero.
const defaultSessionExpiry = 7 * 24 * time.Hour

// sessionTokenBytes is the number of random bytes in a session token before
// hex encoding. 32 bytes = 256 bits of entropy, matching the legacy
// password.SessionService contract.
const sessionTokenBytes = 32

// SessionExpiryConfig lets the caller override the default session TTL. A zero
// Duration means use defaultSessionExpiry.
type SessionExpiryConfig struct {
	Duration time.Duration
}

// IssueSessionRequest captures the inputs to creating a new session.
// WorkspaceUserID/WorkspaceID are optional — pass empty strings when a user
// has not yet been bound to a workspace.
type IssueSessionRequest struct {
	UserID          string
	WorkspaceUserID string
	WorkspaceID     string
}

// IssueSessionResponse contains the issued token and the resolved session row.
type IssueSessionResponse struct {
	Token           string
	SessionID       string
	ExpiresAtUnixMs int64
	WorkspaceUserID string
	WorkspaceID     string
}

// IssueSessionRepositories is the write-side of session lifecycle.
type IssueSessionRepositories struct {
	Session sessionpb.SessionDomainServiceServer
}

// IssueSessionServices groups infrastructure deps. No AuthorizationService —
// this use case establishes identity.
type IssueSessionServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
	IDService          ports.IDService
	Expiry             SessionExpiryConfig
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

// Execute creates a session row and returns the opaque bearer token.
func (uc *IssueSessionUseCase) Execute(
	ctx context.Context,
	req *IssueSessionRequest,
) (*IssueSessionResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *IssueSessionUseCase) executeWithTransaction(
	ctx context.Context,
	req *IssueSessionRequest,
) (*IssueSessionResponse, error) {
	var out *IssueSessionResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translated := contextutil.GetTranslatedMessageWithContext(
				txCtx, uc.services.TranslationService,
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
	req *IssueSessionRequest,
) (*IssueSessionResponse, error) {
	token, err := generateSessionToken()
	if err != nil {
		return nil, err
	}

	sessionID := ""
	if uc.services.IDService != nil {
		sessionID = uc.services.IDService.GenerateID()
	}

	ttl := defaultSessionExpiry
	if uc.services.Expiry.Duration > 0 {
		ttl = uc.services.Expiry.Duration
	}
	expiresAt := time.Now().Add(ttl).UnixMilli()

	data := &sessionpb.Session{
		Id:        sessionID,
		UserId:    req.UserID,
		Token:     token,
		ExpiresAt: expiresAt,
		Active:    true,
	}
	if req.WorkspaceUserID != "" {
		wsu := req.WorkspaceUserID
		data.WorkspaceUserId = &wsu
	}
	if req.WorkspaceID != "" {
		ws := req.WorkspaceID
		data.WorkspaceId = &ws
	}

	resp, err := uc.repositories.Session.CreateSession(ctx, &sessionpb.CreateSessionRequest{Data: data})
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"auth.errors.issue_session_failed", "Failed to issue session [DEFAULT]"))
	}
	created := resp.Data[0]

	out := &IssueSessionResponse{
		Token:           created.Token,
		SessionID:       created.Id,
		ExpiresAtUnixMs: created.ExpiresAt,
	}
	if created.WorkspaceUserId != nil {
		out.WorkspaceUserID = *created.WorkspaceUserId
	}
	if created.WorkspaceId != nil {
		out.WorkspaceID = *created.WorkspaceId
	}
	return out, nil
}

func (uc *IssueSessionUseCase) validateInput(ctx context.Context, req *IssueSessionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"auth.validation.request_required", "Session issuance request is required [DEFAULT]"))
	}
	if req.UserID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
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
