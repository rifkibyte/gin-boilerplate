package http

import (
	"errors"

	"github.com/rifkibyte/gin-boilerplate/internal/user"
	apperrors "github.com/rifkibyte/gin-boilerplate/pkg/errors"
)

// toAppError maps this feature's sentinel errors to transport-level errors.
//
// Unknown errors pass through untouched and become a 500.
func toAppError(err error) error {
	switch {
	case errors.Is(err, user.ErrNotFound):
		return apperrors.NotFound("user.not_found", "user not found")
	case errors.Is(err, user.ErrEmailTaken):
		return apperrors.Conflict("user.email_taken", "email already registered")
	default:
		return err
	}
}
