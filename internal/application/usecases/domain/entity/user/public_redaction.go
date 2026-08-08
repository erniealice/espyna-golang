package user

import userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"

// redactPublicUserResponseFields returns a shallow clone with sensitive public fields removed.
func redactPublicUserResponseFields(user *userpb.User) *userpb.User {
	if user == nil {
		return nil
	}

	clone := *user
	clone.PasswordHash = ""
	clone.PasswordResetToken = nil
	clone.PasswordResetExpires = nil
	clone.FailedLoginAttempts = 0
	clone.LockedUntil = nil

	return &clone
}

// redactPublicUserResponseData returns a shallow-cloned and sanitized response user list.
func redactPublicUserResponseData(users []*userpb.User) []*userpb.User {
	if users == nil {
		return nil
	}

	redacted := make([]*userpb.User, len(users))
	for i, user := range users {
		redacted[i] = redactPublicUserResponseFields(user)
	}

	return redacted
}

// redactPublicUserListPageDataResponse clones the repository response before
// replacing its user list so shared repository results remain intact.
func redactPublicUserListPageDataResponse(resp *userpb.GetUserListPageDataResponse) *userpb.GetUserListPageDataResponse {
	if resp == nil {
		return nil
	}

	clone := *resp
	clone.UserList = redactPublicUserResponseData(resp.UserList)
	return &clone
}

// redactPublicUserItemPageDataResponse clones the repository response before
// replacing its user so shared repository results remain intact.
func redactPublicUserItemPageDataResponse(resp *userpb.GetUserItemPageDataResponse) *userpb.GetUserItemPageDataResponse {
	if resp == nil {
		return nil
	}

	clone := *resp
	clone.User = redactPublicUserResponseFields(resp.User)
	return &clone
}
