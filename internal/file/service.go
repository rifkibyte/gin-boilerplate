package file

import (
	"context"
	"fmt"
	"io"
	"strings"

	"go.uber.org/zap"

	"github.com/rifkibyte/gin-boilerplate/pkg/id"
)

// UploadCommand is the input to Upload.
type UploadCommand struct {
	UserID      string
	Filename    string
	ContentType string
	Size        int64
	Content     io.Reader
}

// Service contains business logic for files.
type Service struct {
	repo    FileRepository
	storage Storage
	logger  *zap.Logger
}

// NewService constructs a Service.
func NewService(repo FileRepository, storage Storage, logger *zap.Logger) *Service {
	return &Service{repo: repo, storage: storage, logger: logger}
}

var allowedTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"application/pdf": true,
	"text/plain":      true,
}

const maxFileSize = 10 * 1024 * 1024

// Upload validates and stores a file.
func (s *Service) Upload(ctx context.Context, cmd UploadCommand) (*File, error) {
	if cmd.Size <= 0 || cmd.Size > maxFileSize {
		return nil, ErrTooLarge
	}
	if !allowedTypes[strings.TrimSpace(cmd.ContentType)] {
		return nil, ErrInvalidType
	}

	path, err := s.storage.Save(ctx, cmd.Filename, cmd.Content)
	if err != nil {
		return nil, fmt.Errorf("file: storage save: %w", err)
	}

	f := &File{
		ID:          id.NewV7(),
		UserID:      cmd.UserID,
		Filename:    cmd.Filename,
		ContentType: cmd.ContentType,
		Size:        cmd.Size,
		Path:        path,
	}
	if err := s.repo.Save(ctx, f); err != nil {
		_ = s.storage.Delete(ctx, path)
		return nil, err
	}

	if s.logger != nil {
		s.logger.Info("file uploaded", zap.String("file_id", f.ID), zap.String("user_id", f.UserID))
	}

	return f, nil
}

// Get returns file metadata.
func (s *Service) Get(ctx context.Context, id string) (*File, error) {
	return s.repo.FindByID(ctx, id)
}

// ListByUser returns files for a user.
func (s *Service) ListByUser(ctx context.Context, userID string) ([]*File, error) {
	return s.repo.ListByUser(ctx, userID)
}
