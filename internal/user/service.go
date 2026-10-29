package user

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/rifkibyte/gin-boilerplate/pkg/id"
)

// CreateUserCommand is the input to Create.
type CreateUserCommand struct {
	Email    string
	Password string
}

// Service contains the business logic for users.
type Service struct {
	reader UserReader
	writer UserWriter
	logger *zap.Logger
}

// NewService constructs a Service.
func NewService(reader UserReader, writer UserWriter, logger *zap.Logger) *Service {
	return &Service{
		reader: reader,
		writer: writer,
		logger: logger,
	}
}

// Create registers a new user with the RoleUser role.
func (s *Service) Create(ctx context.Context, cmd CreateUserCommand) (*User, error) {
	existing, err := s.reader.FindByEmail(ctx, cmd.Email)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(cmd.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	u := &User{
		ID:       id.NewV7(),
		Email:    cmd.Email,
		Password: string(hashed),
		Role:     RoleUser,
	}
	if err := s.writer.Save(ctx, u); err != nil {
		return nil, err
	}

	if s.logger != nil {
		s.logger.Info("user created", zap.String("user_id", u.ID), zap.String("email", u.Email))
	}

	return u, nil
}

// List returns all users.
func (s *Service) List(ctx context.Context) ([]*User, error) {
	return s.reader.FindByRole(ctx, "")
}
