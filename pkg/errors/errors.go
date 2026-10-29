// Package errors provides the standardized AppError type used across the
// transport layer.
//
// The application is split into layers, and each layer speaks its own error
// dialect:
//
//   - The inner feature package (domain/service) returns feature-scoped sentinel
//     errors, e.g. user.ErrEmailTaken. It never imports net/http and never
//     constructs a status code.
//   - The feature's http package maps those sentinels to an *AppError.
//   - pkg/response renders the *AppError as the standard JSON error envelope.
//
// Note the package name collides with the standard library "errors" package.
// Always import this one as apperrors:
//
//	import apperrors "github.com/rifkibyte/gin-boilerplate/pkg/errors"
package errors

import (
	"fmt"
	"net/http"
)

// AppError carries an HTTP status code and a stable, machine-readable code.
//
// The wrapped cause is available through errors.Is/errors.As but is never
// rendered to a client: internal details (SQL errors, upstream failures) must
// not leak over HTTP.
type AppError struct {
	Status  int
	Code    string
	Message string
	cause   error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	return e.Message
}

// Unwrap returns the wrapped cause so that errors.Is and errors.As can inspect
// it. It returns nil when no cause was supplied.
func (e *AppError) Unwrap() error {
	return e.cause
}

// New builds an AppError with an optional cause.
//
// A nil cause is valid and means "this is the whole story" — for example a
// client asking for a user that does not exist.
func New(
	status int,
	code string,
	message string,
	cause error,
) *AppError {
	return &AppError{
		Status:  status,
		Code:    code,
		Message: message,
		cause:   cause,
	}
}

// BadRequest returns a 400 AppError.
func BadRequest(code, message string) *AppError {
	return New(
		http.StatusBadRequest,
		code,
		message,
		nil,
	)
}

// Unauthorized returns a 401 AppError.
func Unauthorized(code, message string) *AppError {
	return New(
		http.StatusUnauthorized,
		code,
		message,
		nil,
	)
}

// Forbidden returns a 403 AppError.
func Forbidden(code, message string) *AppError {
	return New(
		http.StatusForbidden,
		code,
		message,
		nil,
	)
}

// NotFound returns a 404 AppError.
func NotFound(code, message string) *AppError {
	return New(
		http.StatusNotFound,
		code,
		message,
		nil,
	)
}

// Conflict returns a 409 AppError.
func Conflict(code, message string) *AppError {
	return New(
		http.StatusConflict,
		code,
		message,
		nil,
	)
}

// Internal returns a 500 AppError wrapping cause.
//
// The cause is logged but never shown to the client, so the message must be
// safe for a user to read.
func Internal(code, message string, cause error) *AppError {
	return New(
		http.StatusInternalServerError,
		code,
		message,
		cause,
	)
}

// Wrap attaches a cause to an AppError that already exists.
//
// It returns the same error when cause is nil, so it is safe to call
// unconditionally.
func (e *AppError) Wrap(cause error) *AppError {
	if cause == nil {
		return e
	}

	return &AppError{
		Status:  e.Status,
		Code:    e.Code,
		Message: e.Message,
		cause:   cause,
	}
}

// WithMessage returns a copy of the AppError with a different message.
//
// The wrapped cause is preserved.
func (e *AppError) WithMessage(message string) *AppError {
	return &AppError{
		Status:  e.Status,
		Code:    e.Code,
		Message: message,
		cause:   e.cause,
	}
}

// String renders the AppError for logs. Unlike Error, it includes the status
// and code, which is what an operator needs when grepping logs.
func (e *AppError) String() string {
	if e.cause == nil {
		return fmt.Sprintf("app error %d %s: %s", e.Status, e.Code, e.Message)
	}

	return fmt.Sprintf("app error %d %s: %s: %v", e.Status, e.Code, e.Message, e.cause)
}
