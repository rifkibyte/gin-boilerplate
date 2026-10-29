package auth

import (
	"context"
	"time"

	"github.com/rifkibyte/gin-boilerplate/internal/user"
)

// UserFinder looks up users.
type UserFinder interface {
	FindByEmail(ctx context.Context, email string) (*user.User, error)
	FindByID(ctx context.Context, id string) (*user.User, error)
}

// UserWriter persists users.
type UserWriter interface {
	Save(ctx context.Context, u *user.User) error
}

// TokenIssuer issues and verifies JWTs.
type TokenIssuer interface {
	IssueAccessToken(ctx context.Context, userID, role string) (string, error)
	IssueRefreshToken(ctx context.Context, userID string) (string, error)
	VerifyAccessToken(ctx context.Context, token string) (userID, role string, err error)
	VerifyRefreshToken(ctx context.Context, token string) (userID string, err error)
}

// TokenStore stores refresh tokens with TTL and revocation.
type TokenStore interface {
	Save(ctx context.Context, token, userID string, ttl time.Duration) error
	Get(ctx context.Context, token string) (string, error)
	Delete(ctx context.Context, token string) error
}

// Mailer sends emails.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}