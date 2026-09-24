package user

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/shared/identity"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

type fakeSelfUserRepo struct {
	userpb.UnimplementedUserDomainServiceServer
	rows      []*userpb.User
	requested string
}

func (f *fakeSelfUserRepo) ReadUser(_ context.Context, req *userpb.ReadUserRequest) (*userpb.ReadUserResponse, error) {
	f.requested = req.GetData().GetId()
	return &userpb.ReadUserResponse{Data: f.rows}, nil
}

func selfCtx(userID string) context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: userID})
}

func TestReadSelfDisplayReadsOnlyTheSessionUser(t *testing.T) {
	repo := &fakeSelfUserRepo{rows: []*userpb.User{{Id: "u1", FirstName: " Junrey ", LastName: "Tejas", Active: true}}}
	first, last, err := NewReadSelfDisplayUseCase(ReadSelfDisplayRepositories{User: repo}).Execute(selfCtx("u1"))
	if err != nil || first != "Junrey" || last != "Tejas" {
		t.Fatalf("got (%q, %q, %v), want (Junrey, Tejas, nil)", first, last, err)
	}
	if repo.requested != "u1" {
		t.Fatalf("read user id %q, want the session user u1", repo.requested)
	}
}

func TestReadSelfDisplayWithEmailReadsOnlyTheSessionUser(t *testing.T) {
	repo := &fakeSelfUserRepo{rows: []*userpb.User{{
		Id: "u1", FirstName: " Junrey ", LastName: "Tejas",
		EmailAddress: " junrey.tejas@mmis.edu.ph ", Active: true,
	}}}
	first, last, email, err := NewReadSelfDisplayUseCase(ReadSelfDisplayRepositories{User: repo}).ExecuteWithEmail(selfCtx("u1"))
	if err != nil || first != "Junrey" || last != "Tejas" || email != "junrey.tejas@mmis.edu.ph" {
		t.Fatalf("got (%q, %q, %q, %v), want (Junrey, Tejas, junrey.tejas@mmis.edu.ph, nil)", first, last, email, err)
	}
	if repo.requested != "u1" {
		t.Fatalf("read user id %q, want the session user u1", repo.requested)
	}
}

func TestReadSelfDisplayWithEmailFailsClosed(t *testing.T) {
	cases := map[string]struct {
		ctx  context.Context
		rows []*userpb.User
	}{
		"no session user":       {context.Background(), []*userpb.User{{Id: "u1", FirstName: "A", Active: true}}},
		"inactive user":         {selfCtx("u1"), []*userpb.User{{Id: "u1", FirstName: "A", Active: false}}},
		"adapter returns other": {selfCtx("u1"), []*userpb.User{{Id: "u2", FirstName: "Other", Active: true}}},
		"not found":             {selfCtx("u1"), nil},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &fakeSelfUserRepo{rows: tc.rows}
			first, last, email, err := NewReadSelfDisplayUseCase(ReadSelfDisplayRepositories{User: repo}).ExecuteWithEmail(tc.ctx)
			if err == nil || first != "" || last != "" || email != "" {
				t.Fatalf("got (%q, %q, %q, %v), want an error and no name/email", first, last, email, err)
			}
		})
	}
}

func TestReadSelfDisplayFailsClosed(t *testing.T) {
	cases := map[string]struct {
		ctx  context.Context
		rows []*userpb.User
	}{
		"no session user":       {context.Background(), []*userpb.User{{Id: "u1", FirstName: "A", Active: true}}},
		"inactive user":         {selfCtx("u1"), []*userpb.User{{Id: "u1", FirstName: "A", Active: false}}},
		"adapter returns other": {selfCtx("u1"), []*userpb.User{{Id: "u2", FirstName: "Other", Active: true}}},
		"not found":             {selfCtx("u1"), nil},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &fakeSelfUserRepo{rows: tc.rows}
			first, last, err := NewReadSelfDisplayUseCase(ReadSelfDisplayRepositories{User: repo}).Execute(tc.ctx)
			if err == nil || first != "" || last != "" {
				t.Fatalf("got (%q, %q, %v), want an error and no name", first, last, err)
			}
		})
	}
}
