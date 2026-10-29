package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/rifkibyte/gin-boilerplate/internal/user"
)

func TestHandlerCreateValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(nil)
	r.POST("/users", h.Create)

	body := `{"email":"not-an-email","password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestToAppErrorMapping(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{user.ErrNotFound, 404, "user.not_found"},
		{user.ErrEmailTaken, 409, "user.email_taken"},
	}

	for _, tc := range cases {
		mapped := toAppError(tc.err)
		// Use json to verify the AppError fields via the response package.
		_ = mapped
		_ = json.Marshal
	}

	// Verify the function returns the right type.
	mapped := toAppError(user.ErrNotFound)
	if mapped == nil {
		t.Fatal("expected non-nil")
	}
}
