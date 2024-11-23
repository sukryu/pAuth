package errors

import (
	"fmt"
	"net/http"
)

type StatusError struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	Reason     string `json:"reason,omitempty"`
	RetryAfter int    `json:"retryAfter,omitempty"`
}

func (e *StatusError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("status %d: %s: %s", e.Code, e.Message, e.Reason)
	}
	return fmt.Sprintf("status %d: %s", e.Code, e.Message)
}

func NewStatusError(code int, message string) *StatusError {
	return &StatusError{
		Code:    code,
		Message: message,
	}
}

func (e *StatusError) WithReason(reason string) *StatusError {
	e.Reason = reason
	return e
}

var (
	// Authentication errors
	ErrInvalidCredentials = NewStatusError(http.StatusUnauthorized, "invalid credentials")
	ErrTokenExpired       = NewStatusError(http.StatusUnauthorized, "token expired")
	ErrInvalidToken       = NewStatusError(http.StatusUnauthorized, "invalid token")

	// Authorization errors
	ErrForbidden        = NewStatusError(http.StatusForbidden, "forbidden")
	ErrPermissionDenied = NewStatusError(http.StatusForbidden, "permission denied")

	// Resource errors
	ErrUserNotFound = NewStatusError(http.StatusNotFound, "user not found")
	ErrRoleNotFound = NewStatusError(http.StatusNotFound, "role not found")
	ErrUserExists   = NewStatusError(http.StatusConflict, "user already exists")
	ErrRoleExists   = NewStatusError(http.StatusConflict, "role already exists")

	// Validation errors
	ErrInvalidRequest = NewStatusError(http.StatusBadRequest, "invalid request")
	ErrInvalidInput   = NewStatusError(http.StatusBadRequest, "invalid input")

	// Server errors
	ErrInternal       = NewStatusError(http.StatusInternalServerError, "internal server error")
	ErrNotImplemented = NewStatusError(http.StatusNotImplemented, "not implemented")
)