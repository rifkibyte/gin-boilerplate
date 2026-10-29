package user

import "errors"

var (
	// ErrNotFound is returned when a user does not exist.
	ErrNotFound = errors.New("user not found")
	// ErrEmailTaken is returned when a user with the given email already exists.
	ErrEmailTaken = errors.New("email already registered")
)
