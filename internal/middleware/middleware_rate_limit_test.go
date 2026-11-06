package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimitMiddlewareTripsAtThreshold(t *testing.T) {
	gin.SetMode(gin.TestMode)

	counts := map[string]int64{}
	increment := func(ctx context.Context, key string) (int64, error) {
		counts[key]++
		return counts[key], nil
	}

	r := gin.New()
	r.Use(RateLimitMiddleware(increment, 2, time.Second))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}

func TestRateLimitMiddlewareFailsOpenOnError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	increment := func(ctx context.Context, key string) (int64, error) {
		return 0, context.DeadlineExceeded
	}

	r := gin.New()
	r.Use(RateLimitMiddleware(increment, 1, time.Second))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected fail-open 200, got %d", rec.Code)
	}
}
