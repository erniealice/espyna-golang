//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/database/sqlexec"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

type userSecurityFixture struct {
	listCalls   int
	queryCalls  int
	updateCalls int
	updateData  map[string]any

	queryArgs    []any
	query        string
	queryErr     error
	listErr      error
	listResult   *interfaces.ListResult
	allowUpdate  bool
	updateResult map[string]any
	updateErr    error
}

func (f *userSecurityFixture) Create(context.Context, string, map[string]any) (map[string]any, error) {
	return nil, fmt.Errorf("Create should not be called")
}

func (f *userSecurityFixture) Read(context.Context, string, string) (map[string]any, error) {
	return nil, fmt.Errorf("Read should not be called")
}

func (f *userSecurityFixture) Update(_ context.Context, _ string, id string, data map[string]any) (map[string]any, error) {
	f.updateCalls++
	f.updateData = copyStringAnyMap(data)

	if !f.allowUpdate {
		return nil, fmt.Errorf("Update should not be called")
	}
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	if f.updateResult != nil {
		return f.updateResult, nil
	}
	return map[string]any{"id": id}, nil
}

func (f *userSecurityFixture) Delete(context.Context, string, string) error {
	return fmt.Errorf("Delete should not be called")
}

func (f *userSecurityFixture) HardDelete(context.Context, string, string) error {
	return fmt.Errorf("HardDelete should not be called")
}

func (f *userSecurityFixture) List(context.Context, string, *interfaces.ListParams) (*interfaces.ListResult, error) {
	f.listCalls++
	return f.listResult, f.listErr
}

func (f *userSecurityFixture) Query(context.Context, string, interfaces.QueryBuilder) ([]map[string]any, error) {
	return nil, fmt.Errorf("Query should not be called")
}

func (f *userSecurityFixture) QueryOne(context.Context, string, interfaces.QueryBuilder) (map[string]any, error) {
	return nil, fmt.Errorf("QueryOne should not be called")
}

func copyStringAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}

	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func stringPtr(v string) *string {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}

func (f *userSecurityFixture) GetExecutor(context.Context) sqlexec.DBExecutor {
	return f
}

func (f *userSecurityFixture) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, fmt.Errorf("ExecContext should not be called")
}

func (f *userSecurityFixture) QueryContext(_ context.Context, query string, args ...any) (*sql.Rows, error) {
	f.queryCalls++
	f.query = query
	f.queryArgs = args
	return nil, f.queryErr
}

func (f *userSecurityFixture) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return nil
}

func TestListUsers_UsesGetUserListPageDataDefaultPagination(t *testing.T) {
	t.Parallel()
	t.Helper()

	fx := &userSecurityFixture{
		queryErr: errors.New("query-hit"),
		listErr:  errors.New("list should not be called"),
	}
	repo := NewPostgresUserRepository(fx, entityid.User).(*PostgresUserRepository)

	_, err := repo.ListUsers(context.Background(), &userpb.ListUsersRequest{})
	if err == nil || !strings.Contains(err.Error(), "query-hit") {
		t.Fatalf("ListUsers() error = %v, want wrapped executor error", err)
	}
	if fx.listCalls != 0 {
		t.Fatalf("ListUsers() called dbOps.List() %d times, want 0", fx.listCalls)
	}
	if fx.queryCalls != 1 {
		t.Fatalf("ListUsers() called QueryContext() %d times, want 1", fx.queryCalls)
	}
	if len(fx.queryArgs) != 3 {
		t.Fatalf("ListUsers() query args = %v, want active/limit/page triple", fx.queryArgs)
	}
	if got := fx.queryArgs[0]; got != true {
		t.Fatalf("ListUsers() first query arg = %v, want default active=true", got)
	}
	if got := fx.queryArgs[1]; got != int32(100) {
		t.Fatalf("ListUsers() second query arg = %v, want 100", got)
	}
	if got := fx.queryArgs[2]; got != int32(0) {
		t.Fatalf("ListUsers() third query arg = %v, want 0", got)
	}
}

func TestListUsers_PreservesProvidedPagination(t *testing.T) {
	t.Parallel()
	t.Helper()

	fx := &userSecurityFixture{
		queryErr: errors.New("query-hit"),
	}
	repo := NewPostgresUserRepository(fx, entityid.User).(*PostgresUserRepository)

	_, err := repo.ListUsers(context.Background(), &userpb.ListUsersRequest{
		Pagination: &commonpb.PaginationRequest{
			Limit: 15,
			Method: &commonpb.PaginationRequest_Offset{
				Offset: &commonpb.OffsetPagination{Page: 3},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "query-hit") {
		t.Fatalf("ListUsers() error = %v, want wrapped executor error", err)
	}
	if len(fx.queryArgs) != 3 {
		t.Fatalf("ListUsers() query args = %v, want active/limit/page triple", fx.queryArgs)
	}
	if got := fx.queryArgs[0]; got != true {
		t.Fatalf("ListUsers() first query arg = %v, want default active=true", got)
	}
	if got := fx.queryArgs[1]; got != int32(15) {
		t.Fatalf("ListUsers() second query arg = %v, want 15", got)
	}
	if got := fx.queryArgs[2]; got != int32(30) {
		t.Fatalf("ListUsers() third query arg = %v, want 30", got)
	}
}

func TestGetUserListPageData_ActiveFilterDefaultsAndHonorsExplicitChoice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		filters *commonpb.FilterRequest
		want    bool
	}{
		{name: "default true", want: true},
		{name: "explicit true", filters: activeUserFilter(true), want: true},
		{name: "explicit false", filters: activeUserFilter(false), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fx := &userSecurityFixture{queryErr: errors.New("query-hit")}
			repo := NewPostgresUserRepository(fx, entityid.User).(*PostgresUserRepository)

			_, err := repo.GetUserListPageData(context.Background(), &userpb.GetUserListPageDataRequest{Filters: tt.filters})
			if err == nil || !strings.Contains(err.Error(), "query-hit") {
				t.Fatalf("GetUserListPageData() error = %v, want wrapped executor error", err)
			}
			if strings.Count(fx.query, "active = $") != 1 {
				t.Fatalf("query active predicate count = %d, want 1; query: %s", strings.Count(fx.query, "active = $"), fx.query)
			}
			if len(fx.queryArgs) != 3 {
				t.Fatalf("query args = %v, want active/limit/page triple", fx.queryArgs)
			}
			if got := fx.queryArgs[0]; got != tt.want {
				t.Fatalf("active filter arg = %v, want %v", got, tt.want)
			}
		})
	}
}

func activeUserFilter(value bool) *commonpb.FilterRequest {
	return &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
		Field: "active",
		FilterType: &commonpb.TypedFilter_BooleanFilter{BooleanFilter: &commonpb.BooleanFilter{
			Value: value,
		}},
	}}}
}

func TestGetUserListPageData_RejectsUnknownSortField(t *testing.T) {
	t.Parallel()
	t.Helper()

	fx := &userSecurityFixture{
		queryErr: errors.New("query should not be called"),
	}
	repo := NewPostgresUserRepository(fx, entityid.User).(*PostgresUserRepository)

	_, err := repo.GetUserListPageData(context.Background(), &userpb.GetUserListPageDataRequest{
		Sort: &commonpb.SortRequest{
			Fields: []*commonpb.SortField{{Field: "password_hash", Direction: commonpb.SortDirection_ASC}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "unknown sort column") {
		t.Fatalf("GetUserListPageData() error = %v, want unknown sort field error", err)
	}
	if fx.queryCalls != 0 {
		t.Fatalf("GetUserListPageData() called QueryContext() %d times, want 0", fx.queryCalls)
	}
}

func TestGetUserListPageData_RejectsUnknownFilterField(t *testing.T) {
	t.Parallel()
	t.Helper()

	fx := &userSecurityFixture{
		queryErr: errors.New("query should not be called"),
	}
	repo := NewPostgresUserRepository(fx, entityid.User).(*PostgresUserRepository)

	_, err := repo.GetUserListPageData(context.Background(), &userpb.GetUserListPageDataRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{{
				Field: "password_hash",
				FilterType: &commonpb.TypedFilter_StringFilter{
					StringFilter: &commonpb.StringFilter{
						Value: "x", Operator: commonpb.StringOperator_STRING_EQUALS,
					},
				},
			}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "filter field") {
		t.Fatalf("GetUserListPageData() error = %v, want unknown filter field error", err)
	}
	if fx.queryCalls != 0 {
		t.Fatalf("GetUserListPageData() called QueryContext() %d times, want 0", fx.queryCalls)
	}
}

func TestUpdateUser_RemovesCredentialColumnsFromUpdateData(t *testing.T) {
	t.Parallel()
	t.Helper()

	fx := &userSecurityFixture{
		allowUpdate: true,
		updateResult: map[string]any{
			"id":         "user-001",
			"first_name": "Ada",
			"last_name":  "Lovelace",
			"active":     true,
		},
	}
	repo := NewPostgresUserRepository(fx, entityid.User).(*PostgresUserRepository)
	_, err := repo.UpdateUser(context.Background(), &userpb.UpdateUserRequest{
		Data: &userpb.User{
			Id:                   "user-001",
			FirstName:            "Ada",
			LastName:             "Lovelace",
			EmailAddress:         "ada@company.test",
			PasswordHash:         "intruder-hash",
			PasswordResetToken:   stringPtr("intruder-token"),
			PasswordResetExpires: int64Ptr(9999),
			FailedLoginAttempts:  99,
			LockedUntil:          int64Ptr(8888),
			Active:               true,
		},
	})
	if err != nil {
		t.Fatalf("UpdateUser() error = %v, want nil", err)
	}

	if fx.updateCalls != 1 {
		t.Fatalf("UpdateUser() passed map to Update() %d times, want 1", fx.updateCalls)
	}

	for _, key := range []string{
		"password_hash",
		"password_reset_token",
		"password_reset_expires",
		"failed_login_attempts",
		"locked_until",
	} {
		if _, ok := fx.updateData[key]; ok {
			t.Fatalf("UpdateUser() update payload should not include %q", key)
		}
	}

	firstName := fx.updateData["first_name"]
	if firstName == nil {
		firstName = fx.updateData["firstName"]
	}
	if firstName != "Ada" {
		t.Fatalf("expected first_name to remain, got %v", firstName)
	}
	lastName := fx.updateData["last_name"]
	if lastName == nil {
		lastName = fx.updateData["lastName"]
	}
	if lastName != "Lovelace" {
		t.Fatalf("expected last_name to remain, got %v", lastName)
	}
	active := fx.updateData["active"]
	if active != true && active != "true" {
		t.Fatalf("expected active to remain, got %v", active)
	}
	emailAddress := fx.updateData["email_address"]
	if emailAddress == nil {
		emailAddress = fx.updateData["emailAddress"]
	}
	if emailAddress != "ada@company.test" {
		t.Fatalf("expected email_address to remain, got %v", emailAddress)
	}
}
