package http

import (
	"errors"
	"net/http"

	"github.com/rifkibyte/gin-boilerplate/internal/file"
	apperrors "github.com/rifkibyte/gin-boilerplate/pkg/errors"
)

func toAppError(err error) error {
	switch {
	case errors.Is(err, file.ErrNotFound):
		return apperrors.NotFound("file.not_found", "file not found")
	case errors.Is(err, file.ErrTooLarge):
		return apperrors.New(http.StatusRequestEntityTooLarge, "file.too_large", "file too large", err)
	case errors.Is(err, file.ErrInvalidType):
		return apperrors.BadRequest("file.invalid_type", "invalid content type")
	default:
		return err
	}
}
