package file

import "errors"

var (
	ErrNotFound     = errors.New("file not found")
	ErrTooLarge     = errors.New("file too large")
	ErrInvalidType  = errors.New("invalid content type")
	ErrUnauthorized = errors.New("unauthorized")
)