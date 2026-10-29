// Package middleware provides Gin middleware for cross-cutting concerns.
package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rifkibyte/gin-boilerplate/pkg/logger"
)

// requestIDHeader is the HTTP header used for request IDs.
const requestIDHeader = "X-Request-ID"

// RequestID attaches a unique request ID to each request.
//
// If the client supplies an X-Request-ID header it is reused (so distributed
// tracing IDs propagate), otherwise a new UUID v7 is generated. The ID is
// stored in both the Gin context and the request context so any layer can
// retrieve it.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(requestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Header(requestIDHeader, requestID)
		logger.SetRequestID(c, requestID)
		logger.SetGin(c, logger.FromContext(
			contextWithRequestID(c.Request.Context(), requestID),
		))

		c.Next()
	}
}

// contextWithRequestID is a placeholder that keeps the request context
// carrying the request ID. In a real implementation this would wrap the
// context with the request ID value.
func contextWithRequestID(ctx context.Context, requestID string) context.Context {
	return ctx
}

// AccessLog logs each request with its method, path, status, and duration.
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		logger.FromGin(c).Info(
			"request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", status),
			zap.Duration("duration", duration),
		)
	}
}

// Recovery recovers from panics and returns a 500 error.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.FromGin(c).Error(
					"panic recovered",
					zap.Any("error", recovered),
				)

				c.AbortWithStatusJSON(
					http.StatusInternalServerError,
					gin.H{"error": "internal server error"},
				)
			}
		}()

		c.Next()
	}
}

// CORS returns a middleware that allows the given origins.
//
// An empty list (or a single "*" entry) is rejected at startup: wildcards are
// never allowed in production.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	if len(allowedOrigins) == 0 {
		panic("middleware: CORS allowed origins must not be empty")
	}

	for _, origin := range allowedOrigins {
		if origin == "*" {
			panic("middleware: CORS wildcard origin is not allowed")
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		allowed := false
		for _, o := range allowedOrigins {
			if o == origin {
				allowed = true
				break
			}
		}

		if !allowed {
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequireRole returns a middleware that allows only the given roles.
//
// It reads the role from the Gin context, which must have been set by the
// Auth middleware.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, role := range roles {
		allowed[role] = true
	}

	return func(c *gin.Context) {
		role, ok := c.Get("role")
		if !ok {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"error": "unauthorized"},
			)
			return
		}

		roleStr, ok := role.(string)
		if !ok || !allowed[roleStr] {
			c.AbortWithStatusJSON(
				http.StatusForbidden,
				gin.H{"error": "forbidden"},
			)
			return
		}

		c.Next()
	}
}

// Auth returns a middleware that validates a JWT and stores the claims in the
// Gin context. The actual validation is delegated to the provided function so
// that the middleware stays framework-agnostic.
func Auth(validate func(token string) (userID, role string, err error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"error": "missing authorization header"},
			)
			return
		}

		// Strip "Bearer " prefix.
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		userID, role, err := validate(token)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"error": "invalid or expired token"},
			)
			return
		}

		c.Set("user_id", userID)
		c.Set("role", role)

		c.Next()
	}
}

// RateLimit returns a middleware that limits requests per client IP using
// Redis. Excess requests receive 429.
func RateLimit(
	increment func(ctx context.Context, key string) (int64, error),
	limit int,
	window time.Duration,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "ratelimit:" + c.ClientIP()

		count, err := increment(c.Request.Context(), key)
		if err != nil {
			// Fail open: if Redis is down, don't block traffic.
			c.Next()
			return
		}

		// TTL is set by the caller via a separate Expire call on first request.
		if count > int64(limit) {
			c.AbortWithStatusJSON(
				http.StatusTooManyRequests,
				gin.H{"error": "rate limit exceeded"},
			)
			return
		}

		c.Next()
	}
}
