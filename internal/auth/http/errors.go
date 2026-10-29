package http

import (
	"errors"

	"github.com/rifkibyte/gin-boilerplate/internal/auth"
	"github.com/rifkibyte/gin-boilerplate/internal/user"
	apperrors "github.com/rifkibyte/gin-boilerplate/pkg/errors"
)

func toAppError(err error) error {
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		return apperrors.Unauthorized("auth.invalid_credentials", "invalid credentials")
	case errors.Is(err, auth.ErrInvalidToken), errors.Is(err, auth.ErrTokenExpired), errors.Is(err, auth.ErrTokenRevoked):
		return apperrors.Unauthorized("auth.invalid_token", "invalid or expired token")
	case errors.Is(err, user.ErrEmailTaken):
		return apperrors.Conflict("user.email_taken", "email already registered")
	case errors.Is(err, user.ErrNotFound):
		return apperrors.NotFound("user.not_found", "user not found")
	default:
		return err
	}
}