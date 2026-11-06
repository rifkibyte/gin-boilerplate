package adapter

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rifkibyte/gin-boilerplate/pkg/id"
)

// LocalStorage stores files on disk.
type LocalStorage struct {
	baseDir string
}

// NewLocalStorage constructs a LocalStorage.
func NewLocalStorage(baseDir string) *LocalStorage {
	return &LocalStorage{baseDir: baseDir}
}

func (s *LocalStorage) Save(_ context.Context, filename string, r io.Reader) (string, error) {
	if err := os.MkdirAll(s.baseDir, 0o755); err != nil {
		return "", fmt.Errorf("storage: mkdir: %w", err)
	}

	name := id.NewV7() + "_" + filepath.Base(filename)
	fullPath := filepath.Join(s.baseDir, name)

	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("storage: create: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, r); err != nil {
		return "", fmt.Errorf("storage: write: %w", err)
	}

	return fullPath, nil
}

func (s *LocalStorage) Delete(_ context.Context, path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage: remove: %w", err)
	}
	return nil
}