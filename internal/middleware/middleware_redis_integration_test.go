//go:build integration

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rifkibyte/gin-boilerplate/pkg/cache"
	"github.com/rifkibyte/gin-boilerplate/testutil"
)

func TestRateLimitMiddlewareIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	rc := testutil.NewRedisContainer(ctx, t)

	c := cache.New(rc.Addr, "", 0)
	defer func() { _ = c.Close() }()

	window := time.Second
	increment := func(ctx context.Context, key string) (int64, error) {
		n, err := c.Increment(ctx, key)
		if err != nil {
			return 0, err
		}
		if n == 1 {
			_ = c.Expire(ctx, key, window)
		}
		return n, err
	}

	r := gin.New()
	r.Use(RateLimitMiddleware(increment, 2, window))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	// First two requests pass.
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// Third request is rate limited.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}

	// Simulate a Redis outage: requests must still succeed (fail open).
	if err := rc.Container.Terminate(ctx); err != nil {
		t.Fatalf("terminating redis: %v", err)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected fail-open 200 after redis stop, got %d", rec.Code)
	}
}
