package user

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/auth"
	"google.golang.org/protobuf/proto"
)

type fakeAuthService struct {
	verifyTokenCalled    bool
	changePasswordCalled bool
	disableCalled        bool
	enableCalled         bool
	adminSetCalled       bool
	revokeCalled         bool
	generateLinkCalled   bool
	isEnabledValue       bool

	VerifyTokenErr    error
	ChangePasswordErr error
	DisableErr        error
	EnableErr         error
	AdminSetErr       error
	GenerateErr       error
	RevokeErr         error
	UpdateEmailErr    error
}

func (f *fakeAuthService) VerifyToken(_ context.Context, _ *authpb.ValidateJwtTokenRequest) (*authpb.ValidateJwtTokenResponse, error) {
	f.verifyTokenCalled = true
	if f.VerifyTokenErr != nil {
		return nil, f.VerifyTokenErr
	}
	return &authpb.ValidateJwtTokenResponse{}, nil
}

func (f *fakeAuthService) IsEnabled() bool {
	return f.isEnabledValue
}

func (f *fakeAuthService) GetProviderName() string {
	return "fake"
}

func (f *fakeAuthService) ChangePassword(_ context.Context, _, _, _ string) error {
	f.changePasswordCalled = true
	return f.ChangePasswordErr
}

func (f *fakeAuthService) DisableUserAtProvider(_ context.Context, _ string) error {
	f.disableCalled = true
	return f.DisableErr
}

func (f *fakeAuthService) EnableUserAtProvider(_ context.Context, _ string) error {
	f.enableCalled = true
	return f.EnableErr
}

func (f *fakeAuthService) AdminSetPassword(_ context.Context, _ string, _ string) error {
	f.adminSetCalled = true
	return f.AdminSetErr
}

func (f *fakeAuthService) GeneratePasswordResetLink(_ context.Context, _ string) (string, error) {
	f.generateLinkCalled = true
	if f.GenerateErr != nil {
		return "", f.GenerateErr
	}
	return "", nil
}

func (f *fakeAuthService) UpdateEmailAtProvider(_ context.Context, _ string, _ string) error {
	return f.UpdateEmailErr
}

func (f *fakeAuthService) RevokeUserTokens(_ context.Context, _ string) error {
	f.revokeCalled = true
	return f.RevokeErr
}

func newFakeAuthService() *fakeAuthService {
	return &fakeAuthService{
		isEnabledValue: true,
	}
}

type fakeUserRepo struct {
	userpb.UnimplementedUserDomainServiceServer

	createResp   *userpb.CreateUserResponse
	readResp     *userpb.ReadUserResponse
	updateResp   *userpb.UpdateUserResponse
	listResp     *userpb.ListUsersResponse
	listPageResp *userpb.GetUserListPageDataResponse
	itemPageResp *userpb.GetUserItemPageDataResponse

	readReq     *userpb.ReadUserRequest
	updateReq   *userpb.UpdateUserRequest
	readErr     error
	updateErr   error
	readCount   int
	updateCount int
}

func (r *fakeUserRepo) CreateUser(_ context.Context, _ *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
	return r.createResp, nil
}

func (r *fakeUserRepo) ReadUser(_ context.Context, req *userpb.ReadUserRequest) (*userpb.ReadUserResponse, error) {
	r.readReq = req
	r.readCount++
	if r.readErr != nil {
		return nil, r.readErr
	}
	return r.readResp, nil
}

func (r *fakeUserRepo) UpdateUser(_ context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	r.updateReq = req
	r.updateCount++
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	return r.updateResp, nil
}

func (r *fakeUserRepo) ListUsers(_ context.Context, _ *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	return r.listResp, nil
}

func (r *fakeUserRepo) GetUserListPageData(_ context.Context, _ *userpb.GetUserListPageDataRequest) (*userpb.GetUserListPageDataResponse, error) {
	return r.listPageResp, nil
}

func (r *fakeUserRepo) GetUserItemPageData(_ context.Context, _ *userpb.GetUserItemPageDataRequest) (*userpb.GetUserItemPageDataResponse, error) {
	return r.itemPageResp, nil
}

func stringPtr(v string) *string {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}

func sensitiveUser() *userpb.User {
	return &userpb.User{
		Id:                   "user-001",
		FirstName:            "Ada",
		LastName:             "Lovelace",
		EmailAddress:         "ada@company.test",
		PasswordHash:         "hashed:secret",
		PasswordResetToken:   stringPtr("reset-token"),
		PasswordResetExpires: int64Ptr(171000),
		FailedLoginAttempts:  7,
		LockedUntil:          int64Ptr(172000),
	}
}

func assertPublicFieldsCleared(t *testing.T, user *userpb.User) {
	t.Helper()

	if user.PasswordHash != "" {
		t.Fatalf("expected PasswordHash to be cleared, got %q", user.PasswordHash)
	}
	if user.PasswordResetToken != nil {
		t.Fatalf("expected PasswordResetToken to be cleared, got %#v", user.PasswordResetToken)
	}
	if user.PasswordResetExpires != nil {
		t.Fatalf("expected PasswordResetExpires to be cleared, got %#v", user.PasswordResetExpires)
	}
	if user.FailedLoginAttempts != 0 {
		t.Fatalf("expected FailedLoginAttempts to be cleared, got %d", user.FailedLoginAttempts)
	}
	if user.LockedUntil != nil {
		t.Fatalf("expected LockedUntil to be cleared, got %#v", user.LockedUntil)
	}
}

func TestRedactPublicUserResponseFields(t *testing.T) {
	base := sensitiveUser()
	source := &userpb.User{
		Id:                   base.Id,
		FirstName:            base.FirstName,
		LastName:             base.LastName,
		EmailAddress:         base.EmailAddress,
		PasswordHash:         base.PasswordHash,
		PasswordResetToken:   base.PasswordResetToken,
		PasswordResetExpires: base.PasswordResetExpires,
		FailedLoginAttempts:  base.FailedLoginAttempts,
		LockedUntil:          base.LockedUntil,
	}

	sourceCopy := *source
	redacted := redactPublicUserResponseFields(source)
	if redacted == nil {
		t.Fatal("expected redacted user, got nil")
	}

	assertPublicFieldsCleared(t, redacted)

	if redacted.Id != sourceCopy.Id || redacted.FirstName != sourceCopy.FirstName || redacted.LastName != sourceCopy.LastName || redacted.EmailAddress != sourceCopy.EmailAddress {
		t.Fatalf("expected ordinary fields to remain unchanged")
	}

	if source.PasswordHash != sourceCopy.PasswordHash {
		t.Fatalf("input was mutated: PasswordHash got cleared")
	}
	if source.GetPasswordResetToken() != "reset-token" {
		t.Fatalf("input was mutated: PasswordResetToken was changed")
	}
	if source.GetPasswordResetExpires() != 171000 {
		t.Fatalf("input was mutated: PasswordResetExpires was changed")
	}
	if source.GetFailedLoginAttempts() != 7 {
		t.Fatalf("input was mutated: FailedLoginAttempts was changed")
	}
	if source.GetLockedUntil() != 172000 {
		t.Fatalf("input was mutated: LockedUntil was changed")
	}
}

func TestRedactPublicUserResponseData_HandlesNilEntries(t *testing.T) {
	first := sensitiveUser()
	redacted := redactPublicUserResponseData([]*userpb.User{
		first,
		nil,
		sensitiveUser(),
	})

	if len(redacted) != 3 {
		t.Fatalf("expected 3 users, got %d", len(redacted))
	}
	if redacted[1] != nil {
		t.Fatalf("expected nil entry to remain nil")
	}
	assertPublicFieldsCleared(t, redacted[0])
	assertPublicFieldsCleared(t, redacted[2])

	if redacted[0] == first {
		t.Fatalf("expected redacted user to be cloned, not aliased to the original")
	}
}

func TestCreateUserUseCase_Execute_RedactsPublicFields(t *testing.T) {
	repo := &fakeUserRepo{
		createResp: &userpb.CreateUserResponse{
			Data: []*userpb.User{
				sensitiveUser(),
			},
		},
	}

	createServices := CreateUserServices{
		Authorizer:       nil,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		IDGenerator:      ports.NewNoOpIDGenerator(),
	}

	useCase := NewCreateUserUseCase(CreateUserRepositories{User: repo}, createServices)
	resp, err := useCase.Execute(context.Background(), &userpb.CreateUserRequest{
		Data: &userpb.User{
			FirstName:    "Ada",
			LastName:     "Lovelace",
			EmailAddress: "ada@company.test",
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if resp == nil || len(resp.Data) != 1 {
		t.Fatalf("expected one response user")
	}
	assertPublicFieldsCleared(t, resp.Data[0])
	assertPublicFieldsCleared(t, repo.createResp.Data[0])
	if resp.Data[0].Id != repo.createResp.Data[0].Id {
		t.Fatalf("expected ordinary fields to be preserved")
	}
	if resp.Data[0].FirstName != repo.createResp.Data[0].FirstName {
		t.Fatalf("expected ordinary fields to be preserved")
	}
	if resp.Data[0].LastName != repo.createResp.Data[0].LastName {
		t.Fatalf("expected ordinary fields to be preserved")
	}
}

func TestReadUserUseCase_Execute_RedactsPublicFields(t *testing.T) {
	repo := &fakeUserRepo{
		readResp: &userpb.ReadUserResponse{
			Data: []*userpb.User{
				sensitiveUser(),
			},
		},
	}

	readServices := ReadUserServices{
		Authorizer:       nil,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
	}

	useCase := NewReadUserUseCase(ReadUserRepositories{User: repo}, readServices)
	resp, err := useCase.Execute(context.Background(), &userpb.ReadUserRequest{
		Data: &userpb.User{
			Id: "user-001",
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if resp == nil || len(resp.Data) != 1 {
		t.Fatalf("expected one response user")
	}
	assertPublicFieldsCleared(t, resp.Data[0])
	assertPublicFieldsCleared(t, repo.readResp.Data[0])
	if resp.Data[0].Id != repo.readResp.Data[0].Id {
		t.Fatalf("expected ordinary fields to be preserved")
	}
	if resp.Data[0].FirstName != repo.readResp.Data[0].FirstName {
		t.Fatalf("expected ordinary fields to be preserved")
	}
}

func TestUpdateUserUseCase_Execute_RedactsPublicFields(t *testing.T) {
	existing := &userpb.User{
		Id:                   "user-001",
		FirstName:            "Stored",
		LastName:             "Profile",
		EmailAddress:         "ada@company.test",
		PasswordHash:         "stored-password-hash",
		PasswordResetToken:   stringPtr("stored-reset-token"),
		PasswordResetExpires: int64Ptr(171000),
		FailedLoginAttempts:  7,
		LockedUntil:          int64Ptr(172000),
	}

	repo := &fakeUserRepo{
		readResp: &userpb.ReadUserResponse{
			Data: []*userpb.User{
				existing,
			},
		},
		updateResp: &userpb.UpdateUserResponse{
			Data: []*userpb.User{
				sensitiveUser(),
			},
		},
	}

	updateServices := UpdateUserServices{
		Authorizer:       nil,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
	}

	useCase := NewUpdateUserUseCase(UpdateUserRepositories{User: repo}, updateServices)
	input := &userpb.User{
		Id:                   "user-001",
		EmailAddress:         "ada@company.test",
		PasswordHash:         "incoming-password",
		PasswordResetToken:   stringPtr("incoming-reset-token"),
		PasswordResetExpires: int64Ptr(999999),
		FailedLoginAttempts:  9,
		LockedUntil:          int64Ptr(999998),
		FirstName:            "Ada",
		LastName:             "Lovelace",
	}
	inputCopy := proto.Clone(input).(*userpb.User)

	resp, err := useCase.Execute(context.Background(), &userpb.UpdateUserRequest{
		Data: input,
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if repo.readCount != 1 {
		t.Fatalf("ReadUser() called %d times, want 1", repo.readCount)
	}
	if repo.updateCount != 1 {
		t.Fatalf("UpdateUser() called %d times, want 1", repo.updateCount)
	}
	if repo.updateReq == nil || repo.updateReq.Data == nil {
		t.Fatalf("expected repository update request")
	}
	if repo.updateReq.Data == input {
		t.Fatalf("expected repository update request to be a clone of input")
	}
	if repo.updateReq.Data.GetPasswordHash() != existing.GetPasswordHash() {
		t.Fatalf("expected PasswordHash to come from existing row")
	}
	if repo.updateReq.Data.GetPasswordResetToken() != existing.GetPasswordResetToken() {
		t.Fatalf("expected PasswordResetToken to come from existing row")
	}
	if repo.updateReq.Data.GetPasswordResetExpires() != existing.GetPasswordResetExpires() {
		t.Fatalf("expected PasswordResetExpires to come from existing row")
	}
	if repo.updateReq.Data.GetFailedLoginAttempts() != existing.GetFailedLoginAttempts() {
		t.Fatalf("expected FailedLoginAttempts to come from existing row")
	}
	if repo.updateReq.Data.GetLockedUntil() != existing.GetLockedUntil() {
		t.Fatalf("expected LockedUntil to come from existing row")
	}
	if repo.updateReq.Data.GetFirstName() != input.GetFirstName() {
		t.Fatalf("expected ordinary field FirstName to remain from input")
	}
	if repo.updateReq.Data.GetLastName() != input.GetLastName() {
		t.Fatalf("expected ordinary field LastName to remain from input")
	}

	assertPublicFieldsCleared(t, resp.Data[0])
	assertPublicFieldsCleared(t, repo.updateResp.Data[0])
	if !proto.Equal(input, inputCopy) {
		t.Fatalf("expected input request to remain unchanged")
	}
}

func TestUpdateUserUseCase_Execute_ReadErrorAbortsUpdate(t *testing.T) {
	repo := &fakeUserRepo{
		readErr: errors.New("read-failed"),
	}

	updateServices := UpdateUserServices{
		Authorizer:       nil,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
	}

	useCase := NewUpdateUserUseCase(UpdateUserRepositories{User: repo}, updateServices)
	_, err := useCase.Execute(context.Background(), &userpb.UpdateUserRequest{
		Data: &userpb.User{
			Id:           "user-001",
			EmailAddress: "ada@company.test",
			FirstName:    "Ada",
		},
	})
	if err == nil {
		t.Fatalf("expected error from ReadUser failure")
	}
	if !strings.Contains(err.Error(), "read-failed") {
		t.Fatalf("expected read failure to be returned, got %v", err)
	}
	if repo.updateCount != 0 {
		t.Fatalf("expected no UpdateUser call on read failure, got %d", repo.updateCount)
	}
}

func TestUpdateUserUseCase_Execute_RejectsMissingExistingRow(t *testing.T) {
	repo := &fakeUserRepo{
		readResp: &userpb.ReadUserResponse{},
	}

	updateServices := UpdateUserServices{
		Authorizer:       nil,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
	}

	useCase := NewUpdateUserUseCase(UpdateUserRepositories{User: repo}, updateServices)
	_, err := useCase.Execute(context.Background(), &userpb.UpdateUserRequest{
		Data: &userpb.User{
			Id:           "user-001",
			EmailAddress: "ada@company.test",
			FirstName:    "Ada",
		},
	})
	if err == nil {
		t.Fatalf("expected not found error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
	if repo.updateCount != 0 {
		t.Fatalf("expected no UpdateUser call when existing row is missing, got %d", repo.updateCount)
	}
}

func TestListUsersUseCase_Execute_RedactsPublicFields(t *testing.T) {
	repo := &fakeUserRepo{
		listResp: &userpb.ListUsersResponse{
			Data: []*userpb.User{
				sensitiveUser(),
				nil,
			},
		},
	}

	listServices := ListUsersServices{
		Authorizer:       nil,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
	}

	useCase := NewListUsersUseCase(ListUsersRepositories{User: repo}, listServices)
	resp, err := useCase.Execute(context.Background(), &userpb.ListUsersRequest{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if resp == nil || len(resp.Data) != 2 {
		t.Fatalf("expected two response users")
	}
	if resp.Data[1] != nil {
		t.Fatalf("expected second response entry to remain nil")
	}
	assertPublicFieldsCleared(t, resp.Data[0])
	if resp.Data[0].Id != repo.listResp.Data[0].Id {
		t.Fatalf("expected ordinary fields to be preserved")
	}
	if resp.Data[0].FirstName != repo.listResp.Data[0].FirstName {
		t.Fatalf("expected ordinary fields to be preserved")
	}
}

func TestGetUserListPageDataUseCase_Execute_RedactsWithoutMutatingRepositoryResponse(t *testing.T) {
	source := sensitiveUser()
	sourceCopy := proto.Clone(source).(*userpb.User)
	repo := &fakeUserRepo{listPageResp: &userpb.GetUserListPageDataResponse{UserList: []*userpb.User{source, nil}, Success: true}}
	services := GetUserListPageDataServices{ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()), Translator: ports.NewNoOpTranslator()}

	resp, err := NewGetUserListPageDataUseCase(GetUserListPageDataRepositories{User: repo}, services).Execute(context.Background(), &userpb.GetUserListPageDataRequest{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp == repo.listPageResp {
		t.Fatal("expected a cloned page-data response")
	}
	if len(resp.UserList) != 2 || resp.UserList[1] != nil {
		t.Fatalf("expected cloned list to preserve entries, got %#v", resp.UserList)
	}
	assertPublicFieldsCleared(t, resp.UserList[0])
	if !proto.Equal(repo.listPageResp.UserList[0], sourceCopy) {
		t.Fatalf("repository-supplied list user was mutated: %#v", repo.listPageResp.UserList[0])
	}
}

func TestGetUserItemPageDataUseCase_Execute_RedactsWithoutMutatingRepositoryResponse(t *testing.T) {
	source := sensitiveUser()
	sourceCopy := proto.Clone(source).(*userpb.User)
	repo := &fakeUserRepo{itemPageResp: &userpb.GetUserItemPageDataResponse{User: source, Success: true}}
	services := GetUserItemPageDataServices{ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()), Translator: ports.NewNoOpTranslator()}

	resp, err := NewGetUserItemPageDataUseCase(GetUserItemPageDataRepositories{User: repo}, services).Execute(context.Background(), &userpb.GetUserItemPageDataRequest{UserId: source.Id})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp == repo.itemPageResp {
		t.Fatal("expected a cloned page-data response")
	}
	assertPublicFieldsCleared(t, resp.User)
	if !proto.Equal(repo.itemPageResp.User, sourceCopy) {
		t.Fatalf("repository-supplied item user was mutated: %#v", repo.itemPageResp.User)
	}
}

func TestEnableUserUseCase_Execute_PreservesTrustedProfileAndNoMutation(t *testing.T) {
	existing := &userpb.User{
		Id:                   "user-001",
		FirstName:            "Stored",
		LastName:             "Profile",
		EmailAddress:         "ada@company.test",
		PasswordHash:         "stored-password-hash",
		PasswordResetToken:   stringPtr("stored-reset-token"),
		PasswordResetExpires: int64Ptr(171000),
		FailedLoginAttempts:  7,
		LockedUntil:          int64Ptr(172000),
	}
	existingCopy := proto.Clone(existing).(*userpb.User)

	repo := &fakeUserRepo{
		readResp: &userpb.ReadUserResponse{
			Data: []*userpb.User{
				existing,
			},
		},
	}

	enableServices := EnableUserServices{
		Authorizer:       nil,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		AuthService:      newFakeAuthService(),
	}
	useCase := NewEnableUserUseCase(EnableUserRepositories{User: repo}, enableServices)

	resp, err := useCase.Execute(context.Background(), &userpb.EnableUserRequest{
		UserId: "user-001",
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !resp.Enabled {
		t.Fatalf("expected Enabled=true response")
	}

	if repo.readCount != 1 {
		t.Fatalf("ReadUser() called %d times, want 1", repo.readCount)
	}
	if repo.updateCount != 1 {
		t.Fatalf("UpdateUser() called %d times, want 1", repo.updateCount)
	}
	if repo.updateReq == nil || repo.updateReq.Data == nil {
		t.Fatalf("expected repository update request")
	}
	if repo.updateReq.Data == existing {
		t.Fatalf("expected update request to use cloned user row")
	}
	if repo.updateReq.Data.GetActive() != true {
		t.Fatalf("expected Active=true in update payload")
	}
	if repo.updateReq.Data.GetFirstName() != existing.GetFirstName() {
		t.Fatalf("expected trusted FirstName to be preserved")
	}
	if repo.updateReq.Data.GetLastName() != existing.GetLastName() {
		t.Fatalf("expected trusted LastName to be preserved")
	}
	if repo.updateReq.Data.GetEmailAddress() != existing.GetEmailAddress() {
		t.Fatalf("expected trusted EmailAddress to be preserved")
	}
	if repo.updateReq.Data.GetPasswordHash() != existing.GetPasswordHash() {
		t.Fatalf("expected PasswordHash to be preserved")
	}
	if !proto.Equal(existing, existingCopy) {
		t.Fatalf("expected existing row returned from repository to remain unchanged")
	}
}

func TestDisableUserUseCase_Execute_PreservesTrustedProfileAndNoMutation(t *testing.T) {
	existing := &userpb.User{
		Id:                   "user-001",
		FirstName:            "Stored",
		LastName:             "Profile",
		EmailAddress:         "ada@company.test",
		PasswordHash:         "stored-password-hash",
		PasswordResetToken:   stringPtr("stored-reset-token"),
		PasswordResetExpires: int64Ptr(171000),
		FailedLoginAttempts:  7,
		LockedUntil:          int64Ptr(172000),
	}
	existingCopy := proto.Clone(existing).(*userpb.User)

	repo := &fakeUserRepo{
		readResp: &userpb.ReadUserResponse{
			Data: []*userpb.User{
				existing,
			},
		},
	}

	disableServices := DisableUserServices{
		Authorizer:       nil,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		AuthService:      newFakeAuthService(),
	}
	useCase := NewDisableUserUseCase(DisableUserRepositories{User: repo}, disableServices)

	resp, err := useCase.Execute(context.Background(), &userpb.DisableUserRequest{
		UserId: "user-001",
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !resp.Disabled {
		t.Fatalf("expected Disabled=true response")
	}

	if repo.readCount != 1 {
		t.Fatalf("ReadUser() called %d times, want 1", repo.readCount)
	}
	if repo.updateCount != 1 {
		t.Fatalf("UpdateUser() called %d times, want 1", repo.updateCount)
	}
	if repo.updateReq == nil || repo.updateReq.Data == nil {
		t.Fatalf("expected repository update request")
	}
	if repo.updateReq.Data == existing {
		t.Fatalf("expected update request to use cloned user row")
	}
	if repo.updateReq.Data.GetActive() != false {
		t.Fatalf("expected Active=false in update payload")
	}
	if repo.updateReq.Data.GetFirstName() != existing.GetFirstName() {
		t.Fatalf("expected trusted FirstName to be preserved")
	}
	if repo.updateReq.Data.GetLastName() != existing.GetLastName() {
		t.Fatalf("expected trusted LastName to be preserved")
	}
	if repo.updateReq.Data.GetEmailAddress() != existing.GetEmailAddress() {
		t.Fatalf("expected trusted EmailAddress to be preserved")
	}
	if repo.updateReq.Data.GetPasswordHash() != existing.GetPasswordHash() {
		t.Fatalf("expected PasswordHash to be preserved")
	}
	if !proto.Equal(existing, existingCopy) {
		t.Fatalf("expected existing row returned from repository to remain unchanged")
	}
}

func TestEnableUserUseCase_Execute_ReadFailureAbortsUpdate(t *testing.T) {
	fxAuth := newFakeAuthService()
	repo := &fakeUserRepo{
		readErr: errors.New("read-failed"),
	}

	enableServices := EnableUserServices{
		Authorizer:       nil,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		AuthService:      fxAuth,
	}
	useCase := NewEnableUserUseCase(EnableUserRepositories{User: repo}, enableServices)

	_, err := useCase.Execute(context.Background(), &userpb.EnableUserRequest{
		UserId: "user-001",
	})
	if err == nil {
		t.Fatalf("expected error on read failure")
	}
	if !strings.Contains(err.Error(), "read-failed") {
		t.Fatalf("expected wrapped read error, got %v", err)
	}
	if repo.updateCount != 0 {
		t.Fatalf("expected no UpdateUser call on read failure, got %d", repo.updateCount)
	}
	if fxAuth.enableCalled || fxAuth.revokeCalled || fxAuth.disableCalled {
		t.Fatalf("expected no provider calls on read failure")
	}
}

func TestEnableUserUseCase_Execute_MissingRowAbortsUpdate(t *testing.T) {
	repo := &fakeUserRepo{
		readResp: &userpb.ReadUserResponse{},
	}

	enableServices := EnableUserServices{
		Authorizer:       nil,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
	}
	useCase := NewEnableUserUseCase(EnableUserRepositories{User: repo}, enableServices)

	_, err := useCase.Execute(context.Background(), &userpb.EnableUserRequest{
		UserId: "user-001",
	})
	if err == nil {
		t.Fatalf("expected not found error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
	if repo.updateCount != 0 {
		t.Fatalf("expected no UpdateUser call when existing row is missing, got %d", repo.updateCount)
	}
}

func TestDisableUserUseCase_Execute_ReadFailureAbortsUpdate(t *testing.T) {
	fxAuth := newFakeAuthService()
	repo := &fakeUserRepo{
		readErr: errors.New("read-failed"),
	}

	disableServices := DisableUserServices{
		Authorizer:       nil,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		AuthService:      fxAuth,
	}
	useCase := NewDisableUserUseCase(DisableUserRepositories{User: repo}, disableServices)

	_, err := useCase.Execute(context.Background(), &userpb.DisableUserRequest{
		UserId: "user-001",
	})
	if err == nil {
		t.Fatalf("expected error on read failure")
	}
	if !strings.Contains(err.Error(), "read-failed") {
		t.Fatalf("expected wrapped read error, got %v", err)
	}
	if repo.updateCount != 0 {
		t.Fatalf("expected no UpdateUser call on read failure, got %d", repo.updateCount)
	}
	if fxAuth.disableCalled || fxAuth.revokeCalled || fxAuth.enableCalled {
		t.Fatalf("expected no provider calls on read failure")
	}
}

func TestDisableUserUseCase_Execute_DBUpdateFailureSkipsProvider(t *testing.T) {
	fxAuth := newFakeAuthService()
	existing := &userpb.User{
		Id:        "user-001",
		FirstName: "Stored",
		Active:    true,
	}
	repo := &fakeUserRepo{
		readResp: &userpb.ReadUserResponse{
			Data: []*userpb.User{
				existing,
			},
		},
		updateErr: errors.New("update-failed"),
	}

	disableServices := DisableUserServices{
		Authorizer:       nil,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		AuthService:      fxAuth,
	}
	useCase := NewDisableUserUseCase(DisableUserRepositories{User: repo}, disableServices)

	_, err := useCase.Execute(context.Background(), &userpb.DisableUserRequest{
		UserId: "user-001",
	})
	if err == nil {
		t.Fatalf("expected update error")
	}
	if !strings.Contains(err.Error(), "update-failed") {
		t.Fatalf("expected wrapped update error, got %v", err)
	}
	if repo.updateCount != 1 {
		t.Fatalf("expected one UpdateUser call, got %d", repo.updateCount)
	}
	if fxAuth.disableCalled || fxAuth.revokeCalled {
		t.Fatalf("expected no provider calls on DB update failure")
	}
}
