package http

import (
	"errors"
	"net/http"
	"testing"

	"github.com/rifkibyte/gin-boilerplate/internal/file"
	apperrors "github.com/rifkibyte/gin-boilerplate/pkg/errors"
)

func TestToAppError(t *testing.T) {
	cases := []struct {
		err        error
		wantStatus int
	}{
		{file.ErrNotFound, http.StatusNotFound},
		{file.ErrTooLarge, http.StatusRequestEntityTooLarge},
		{file.ErrInvalidType, http.StatusBadRequest},
		{errors.New("boom"), 0},
	}

	for _, tc := range cases {
		got := toAppError(tc.err)
		if tc.wantStatus == 0 {
			if got != tc.err {
				t.Fatalf("expected passthrough, got %v", got)
			}
			continue
		}
		appErr, ok := got.(*apperrors.AppError)
		if !ok {
			t.Fatalf("expected *AppError, got %T", got)
		}
		if appErr.Status != tc.wantStatus {
			t.Errorf("status: got %d, want %d", appErr.Status, tc.wantStatus)
		}
	}
}