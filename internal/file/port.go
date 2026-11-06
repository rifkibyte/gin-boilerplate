package file

import (
	"context"
	"io"
)

// FileRepository persists file metadata.
type FileRepository interface {
	Save(ctx context.Context, f *File) error
	FindByID(ctx context.Context, id string) (*File, error)
	ListByUser(ctx context.Context, userID string) ([]*File, error)
}

// Storage stores file bytes.
type Storage interface {
	Save(ctx context.Context, filename string, r io.Reader) (string, error)
	Delete(ctx context.Context, path string) error
}
