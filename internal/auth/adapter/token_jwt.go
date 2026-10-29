package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/rifkibyte/gin-boilerplate/internal/auth"
)

// JWTIssuer issues and verifies HS256 JWTs.
type JWTIssuer struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewJWTIssuer constructs a JWTIssuer.
func NewJWTIssuer(secret string, accessTTL, refreshTTL time.Duration) *JWTIssuer {
	return &JWTIssuer{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

type jwtClaims struct {
	UserID string `json:"uid"`
	Role   string `json:"role,omitempty"`
	Type   string `json:"typ"`
	jwt.RegisteredClaims
}

func (j *JWTIssuer) IssueAccessToken(_ context.Context, userID, role string) (string, error) {
	return j.issue(userID, role, "access", j.accessTTL)
}

func (j *JWTIssuer) IssueRefreshToken(_ context.Context, userID string) (string, error) {
	return j.issue(userID, "", "refresh", j.refreshTTL)
}

func (j *JWTIssuer) issue(userID, role, typ string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := jwtClaims{
		UserID: userID,
		Role:   role,
		Type:   typ,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

func (j *JWTIssuer) VerifyAccessToken(_ context.Context, tokenStr string) (string, string, error) {
	claims, err := j.verify(tokenStr, "access")
	if err != nil {
		return "", "", err
	}
	return claims.UserID, claims.Role, nil
}

func (j *JWTIssuer) VerifyRefreshToken(_ context.Context, tokenStr string) (string, error) {
	claims, err := j.verify(tokenStr, "refresh")
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

func (j *JWTIssuer) verify(tokenStr, expectedType string) (*jwtClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return j.secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, auth.ErrInvalidToken
	}
	if claims.Type != expectedType {
		return nil, auth.ErrInvalidToken
	}

	return claims, nil
}