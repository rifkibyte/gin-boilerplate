// Package response renders uniform JSON success and error envelopes.
//
// Every endpoint returns the same shapes so that clients can rely on them:
//
//	success: {"data": {...}}
//	failure: {"error": {"code": "...", "message": "..."}}
package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	apperrors "github.com/rifkibyte/gin-boilerplate/pkg/errors"
	"github.com/rifkibyte/gin-boilerplate/pkg/logger"
)

// SuccessEnvelope is the standard wrapper for a successful response.
type SuccessEnvelope struct {
	Data any `json:"data"`
}

// ErrorBody is the standard error payload.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorEnvelope is the standard wrapper for a failed response.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// OK writes a 200 response with the given payload.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, SuccessEnvelope{Data: data})
}

// Created writes a 201 response with the given payload.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, SuccessEnvelope{Data: data})
}

// NoContent writes a 204 response with no body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Error writes an error response using the given status and message.
//
// Prefer FromError for anything derived from an error value: it handles the
// AppError contract and keeps the response shape consistent.
func Error(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, ErrorEnvelope{
		Error: ErrorBody{
			Code:    defaultCode(status),
			Message: message,
		},
	})
}

// FromError renders err as the standard error envelope.
//
// When err is an *AppError its status, code, and message are used. Any other
// error is treated as an internal failure: it is logged with its full cause and
// reported to the client as a generic 500, so internals never leak.
func FromError(c *gin.Context, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		c.AbortWithStatusJSON(appErr.Status, ErrorEnvelope{
			Error: ErrorBody{
				Code:    appErr.Code,
				Message: appErr.Message,
			},
		})

		return
	}

	logger.FromGin(c).Error("unhandled error", zap.Error(err))

	c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorEnvelope{
		Error: ErrorBody{
			Code:    "internal_error",
			Message: "internal server error",
		},
	})
}

// defaultCode maps a status code to a fallback machine-readable code.
func defaultCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusTooManyRequests:
		return "rate_limited"
	default:
		return "error"
	}
}
