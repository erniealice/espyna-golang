package auth

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	sessionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/session"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// keyEchoTranslator returns the translator KEY as the message so tests can
// assert which error path was taken.
type keyEchoTranslator struct{}

func newKeyEchoTranslator() ports.Translator { return keyEchoTranslator{} }

func (keyEchoTranslator) Get(_ context.Context, _, key string, _ ...any) string {
	return key
}

func (keyEchoTranslator) GetWithDefault(_ context.Context, _, key, _ string, _ ...any) string {
	return key
}

// fakeSessionRepo is a minimal stub for SessionDomainServiceServer that returns
// canned proto responses + records the last CreateSession/UpdateSession calls so
// tests can assert what the use case wrote.
type fakeSessionRepo struct {
	sessionpb.UnimplementedSessionDomainServiceServer

	readResp *sessionpb.ReadSessionResponse
	readErr  error

	createResp *sessionpb.CreateSessionResponse
	createErr  error

	updateResp *sessionpb.UpdateSessionResponse
	updateErr  error

	lastCreate *sessionpb.CreateSessionRequest
	lastUpdate *sessionpb.UpdateSessionRequest
	createN    int
	updateN    int
}

func (f *fakeSessionRepo) ReadSession(_ context.Context, _ *sessionpb.ReadSessionRequest) (*sessionpb.ReadSessionResponse, error) {
	return f.readResp, f.readErr
}

func (f *fakeSessionRepo) CreateSession(_ context.Context, req *sessionpb.CreateSessionRequest) (*sessionpb.CreateSessionResponse, error) {
	f.createN++
	f.lastCreate = req
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResp != nil {
		return f.createResp, nil
	}
	// Default: echo the input row back with Success=true so callers can assert downstream fields.
	return &sessionpb.CreateSessionResponse{
		Data:    []*sessionpb.Session{req.GetData()},
		Success: true,
	}, nil
}

func (f *fakeSessionRepo) UpdateSession(_ context.Context, req *sessionpb.UpdateSessionRequest) (*sessionpb.UpdateSessionResponse, error) {
	f.updateN++
	f.lastUpdate = req
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	if f.updateResp != nil {
		return f.updateResp, nil
	}
	return &sessionpb.UpdateSessionResponse{Success: true}, nil
}

// fakeUserRepo is a minimal stub for UserDomainServiceServer.
type fakeUserRepo struct {
	userpb.UnimplementedUserDomainServiceServer

	readResp *userpb.ReadUserResponse
	readErr  error
}

func (f *fakeUserRepo) ReadUser(_ context.Context, _ *userpb.ReadUserRequest) (*userpb.ReadUserResponse, error) {
	return f.readResp, f.readErr
}

func stringPtr(s string) *string { return &s }
