package http

import (
	"errors"
	"testing"

	"github.com/rifkibyte/gin-boilerplate/internal/user"
	apperrors "github.com/rifkibyte/gin-boilerplate/pkg/errors"
)

func TestToAppError(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"not found", user.ErrNotFound, 404, "user.not_found"},
		{"email taken", user.ErrEmailTaken, 409, "user.email_taken"},
		{"unknown", errors.New("boom"), 0, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := toAppError(tc.err)

			if tc.wantStatus == 0 {
				// Unknown errors pass through.
				if got != tc.err {
					t.Fatalf("expected passthrough, got %v", got)
				}
				return
			}

			appErr, ok := got.(*apperrors.AppError)
			if !ok {
				t.Fatalf("expected *AppError, got %T", got)
			}
			if appErr.Status != tc.wantStatus {
				t.Errorf("status: got %d, want %d", appErr.Status, tc.wantStatus)
			}
			if appErr.Code != tc.wantCode {
				t.Errorf("code: got %s, want %s", appErr.Code, tc.wantCode)
			}
		})
	}
}
