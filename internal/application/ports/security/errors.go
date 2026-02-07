package security

import "fmt"

// AuthorizationError represents authorization-specific errors
type AuthorizationError struct {
	Code    AuthorizationErrorCode
	Message string
	UserID  string
	Details map[string]any
}

// Error implements the error interface
func (e *AuthorizationError) Error() string {
	if e.UserID != "" {
		return fmt.Sprintf("authorization error [%s] for user %s: %s", e.Code, e.UserID, e.Message)
	}
	return fmt.Sprintf("authorization error [%s]: %s", e.Code, e.Message)
}

// AuthorizationErrorCode represents different types of authorization errors
type AuthorizationErrorCode string

const (
	// Permission denied errors
	AuthErrCodePermissionDenied      AuthorizationErrorCode = "PERMISSION_DENIED"
	AuthErrCodeInsufficientRole      AuthorizationErrorCode = "INSUFFICIENT_ROLE"
	AuthErrCodeWorkspaceAccessDenied AuthorizationErrorCode = "WORKSPACE_ACCESS_DENIED"

	// User/context errors
	AuthErrCodeUserNotFound         AuthorizationErrorCode = "USER_NOT_FOUND"
	AuthErrCodeUserNotAuthenticated AuthorizationErrorCode = "USER_NOT_AUTHENTICATED"
	AuthErrCodeInvalidUserID        AuthorizationErrorCode = "INVALID_USER_ID"

	// Workspace errors
	AuthErrCodeWorkspaceNotFound  AuthorizationErrorCode = "WORKSPACE_NOT_FOUND"
	AuthErrCodeInvalidWorkspaceID AuthorizationErrorCode = "INVALID_WORKSPACE_ID"

	// Permission/role errors
	AuthErrCodeInvalidPermission  AuthorizationErrorCode = "INVALID_PERMISSION"
	AuthErrCodeRoleNotFound       AuthorizationErrorCode = "ROLE_NOT_FOUND"
	AuthErrCodePermissionNotFound AuthorizationErrorCode = "PERMISSION_NOT_FOUND"

	// Provider errors
	AuthErrCodeProviderUnavailable AuthorizationErrorCode = "PROVIDER_UNAVAILABLE"
	AuthErrCodeProviderError       AuthorizationErrorCode = "PROVIDER_ERROR"
	AuthErrCodeConfigurationError  AuthorizationErrorCode = "CONFIGURATION_ERROR"

	// System errors
	AuthErrCodeServiceDisabled AuthorizationErrorCode = "SERVICE_DISABLED"
	AuthErrCodeInternalError   AuthorizationErrorCode = "INTERNAL_ERROR"
)

// NewAuthorizationError creates a new authorization error
func NewAuthorizationError(code AuthorizationErrorCode, message string, userID string) *AuthorizationError {
	return &AuthorizationError{
		Code:    code,
		Message: message,
		UserID:  userID,
		Details: make(map[string]any),
	}
}

// WithDetails adds additional context to the error
func (e *AuthorizationError) WithDetails(key string, value any) *AuthorizationError {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}
	e.Details[key] = value
	return e
}

// IsPermissionDenied checks if the error is a permission denied error
func (e *AuthorizationError) IsPermissionDenied() bool {
	return e.Code == AuthErrCodePermissionDenied ||
		e.Code == AuthErrCodeInsufficientRole ||
		e.Code == AuthErrCodeWorkspaceAccessDenied
}

// IsUserError checks if the error is related to user context
func (e *AuthorizationError) IsUserError() bool {
	return e.Code == AuthErrCodeUserNotFound ||
		e.Code == AuthErrCodeUserNotAuthenticated ||
		e.Code == AuthErrCodeInvalidUserID
}

// IsSystemError checks if the error is a system/configuration error
func (e *AuthorizationError) IsSystemError() bool {
	return e.Code == AuthErrCodeProviderUnavailable ||
		e.Code == AuthErrCodeProviderError ||
		e.Code == AuthErrCodeConfigurationError ||
		e.Code == AuthErrCodeServiceDisabled ||
		e.Code == AuthErrCodeInternalError
}

// Predefined error constructors for common scenarios

// ErrPermissionDenied creates a permission denied error
func ErrPermissionDenied(userID, permission string) *AuthorizationError {
	return NewAuthorizationError(
		AuthErrCodePermissionDenied,
		fmt.Sprintf("user does not have permission: %s", permission),
		userID,
	).WithDetails("permission", permission)
}

// ErrWorkspaceAccessDenied creates a workspace access denied error
func ErrWorkspaceAccessDenied(userID, workspaceID string) *AuthorizationError {
	return NewAuthorizationError(
		AuthErrCodeWorkspaceAccessDenied,
		fmt.Sprintf("user does not have access to workspace: %s", workspaceID),
		userID,
	).WithDetails("workspace_id", workspaceID)
}

// ErrUserNotAuthenticated creates a user not authenticated error
func ErrUserNotAuthenticated() *AuthorizationError {
	return NewAuthorizationError(
		AuthErrCodeUserNotAuthenticated,
		"user is not authenticated",
		"",
	)
}

// ErrInsufficientRole creates an insufficient role error
func ErrInsufficientRole(userID string, requiredRole string) *AuthorizationError {
	return NewAuthorizationError(
		AuthErrCodeInsufficientRole,
		fmt.Sprintf("user does not have required role: %s", requiredRole),
		userID,
	).WithDetails("required_role", requiredRole)
}

// ErrProviderUnavailable creates a provider unavailable error
func ErrProviderUnavailable(providerName string) *AuthorizationError {
	return NewAuthorizationError(
		AuthErrCodeProviderUnavailable,
		fmt.Sprintf("authorization provider unavailable: %s", providerName),
		"",
	).WithDetails("provider", providerName)
}

// ErrServiceDisabled creates a service disabled error
func ErrServiceDisabled() *AuthorizationError {
	return NewAuthorizationError(
		AuthErrCodeServiceDisabled,
		"authorization service is disabled",
		"",
	)
}

// ErrAccessDenied is an alias for ErrPermissionDenied for backward compatibility
func ErrAccessDenied(userID string) *AuthorizationError {
	return NewAuthorizationError(
		AuthErrCodePermissionDenied,
		"access denied",
		userID,
	)
}
