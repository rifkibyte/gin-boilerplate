package errors

import (
	"errors"
	"testing"
)

func TestAppErrorUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := Internal("internal_error", "something broke", cause)

	if !errors.Is(err, cause) {
		t.Fatal("errors.Is should find the wrapped cause")
	}

	unwrapped := errors.Unwrap(err)
	if unwrapped != cause {
		t.Fatalf("expected %v, got %v", cause, unwrapped)
	}
}

func TestAppErrorNoCause(t *testing.T) {
	err := NotFound("user.not_found", "user not found")

	if errors.Unwrap(err) != nil {
		t.Fatal("expected nil cause for NotFound")
	}

	if err.Error() != "user not found" {
		t.Fatalf("unexpected message: %s", err.Error())
	}
}

func TestAppErrorWrap(t *testing.T) {
	original := NotFound("user.not_found", "user not found")
	cause := errors.New("db timeout")

	wrapped := original.Wrap(cause)
	if !errors.Is(wrapped, cause) {
		t.Fatal("wrapped error should carry the cause")
	}

	// Original should be unchanged.
	if errors.Unwrap(original) != nil {
		t.Fatal("original should not be mutated")
	}
}
