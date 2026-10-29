package auth

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/rifkibyte/gin-boilerplate/internal/user"
	"github.com/rifkibyte/gin-boilerplate/pkg/id"
)

// RegisterCommand is the input to Register.
type RegisterCommand struct {
	Email    string
	Password string
}

// LoginCommand is the input to Login.
type LoginCommand struct {
	Email    string
	Password string
}

// Service contains the business logic for authentication.
type Service struct {
	users  UserFinder
	writer UserWriter
	tokens TokenIssuer
	store  TokenStore
	mailer Mailer
	logger *zap.Logger
}

// NewService constructs a Service.
func NewService(
	users UserFinder,
	writer UserWriter,
	tokens TokenIssuer,
	store TokenStore,
	mailer Mailer,
	logger *zap.Logger,
) *Service {
	return &Service{
		users:  users,
		writer: writer,
		tokens: tokens,
		store:  store,
		mailer: mailer,
		logger: logger,
	}
}

// Register creates a new user with the RoleUser role.
func (s *Service) Register(ctx context.Context, cmd RegisterCommand) (*user.User, error) {
	existing, err := s.users.FindByEmail(ctx, cmd.Email)
	if err != nil && !errors.Is(err, user.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, user.ErrEmailTaken
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(cmd.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	u := &user.User{
		ID:       id.NewV7(),
		Email:    cmd.Email,
		Password: string(hashed),
		Role:     user.RoleUser,
	}
	if err := s.writer.Save(ctx, u); err != nil {
		return nil, err
	}

	if s.mailer != nil {
		_ = s.mailer.Send(ctx, u.Email, "Welcome", "Welcome to the platform")
	}

	if s.logger != nil {
		s.logger.Info("user registered", zap.String("user_id", u.ID), zap.String("email", u.Email))
	}

	return u, nil
}

// Login authenticates a user and issues tokens.
func (s *Service) Login(ctx context.Context, cmd LoginCommand) (*TokenPair, error) {
	u, err := s.users.FindByEmail(ctx, cmd.Email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(cmd.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := s.tokens.IssueAccessToken(ctx, u.ID, string(u.Role))
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokens.IssueRefreshToken(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	if s.store != nil {
		if err := s.store.Save(ctx, refreshToken, u.ID, 7*24*time.Hour); err != nil {
			return nil, err
		}
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900,
	}, nil
}

// Refresh rotates a refresh token.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	userID, err := s.tokens.VerifyRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if s.store != nil {
		storedUserID, err := s.store.Get(ctx, refreshToken)
		if err != nil || storedUserID != userID {
			return nil, ErrTokenRevoked
		}
	}

	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	accessToken, err := s.tokens.IssueAccessToken(ctx, u.ID, string(u.Role))
	if err != nil {
		return nil, err
	}

	newRefresh, err := s.tokens.IssueRefreshToken(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	if s.store != nil {
		_ = s.store.Delete(ctx, refreshToken)
		_ = s.store.Save(ctx, newRefresh, u.ID, 7*24*time.Hour)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
		ExpiresIn:    900,
	}, nil
}

// Logout revokes a refresh token.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if s.store != nil {
		return s.store.Delete(ctx, refreshToken)
	}
	return nil
}