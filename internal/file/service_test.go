package file

import (
	"context"
	"io"
	"strings"
	"testing"
)

type fakeFileRepo struct {
	saved []*File
}

func newFakeFileRepo() *fakeFileRepo {
	return &fakeFileRepo{saved: []*File{}}
}

func (f *fakeFileRepo) Save(_ context.Context, file *File) error {
	f.saved = append(f.saved, file)
	return nil
}

func (f *fakeFileRepo) FindByID(_ context.Context, id string) (*File, error) {
	for _, item := range f.saved {
		if item.ID == id {
			return item, nil
		}
	}
	return nil, ErrNotFound
}

func (f *fakeFileRepo) ListByUser(_ context.Context, userID string) ([]*File, error) {
	var out []*File
	for _, item := range f.saved {
		if item.UserID == userID {
			out = append(out, item)
		}
	}
	return out, nil
}

type fakeStorage struct {
	saved []string
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{saved: []string{}}
}

func (s *fakeStorage) Save(_ context.Context, filename string, r io.Reader) (string, error) {
	_, _ = io.Copy(io.Discard, r)
	path := "fake://" + filename
	s.saved = append(s.saved, path)
	return path, nil
}

func (s *fakeStorage) Delete(_ context.Context, path string) error {
	return nil
}

func TestServiceUpload(t *testing.T) {
	repo := newFakeFileRepo()
	storage := newFakeStorage()
	svc := NewService(repo, storage, nil)

	f, err := svc.Upload(context.Background(), UploadCommand{
		UserID:      "user-1",
		Filename:    "test.txt",
		ContentType: "text/plain",
		Size:        10,
		Content:     strings.NewReader("hello"),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if f.ContentType != "text/plain" {
		t.Errorf("unexpected content type: %s", f.ContentType)
	}
	if len(repo.saved) != 1 {
		t.Errorf("expected 1 saved, got %d", len(repo.saved))
	}
}

func TestServiceUploadInvalidType(t *testing.T) {
	svc := NewService(newFakeFileRepo(), newFakeStorage(), nil)
	_, err := svc.Upload(context.Background(), UploadCommand{
		UserID:      "user-1",
		Filename:    "test.exe",
		ContentType: "application/x-msdownload",
		Size:        10,
		Content:     strings.NewReader("hello"),
	})
	if err != ErrInvalidType {
		t.Fatalf("expected ErrInvalidType, got %v", err)
	}
}

func TestServiceUploadTooLarge(t *testing.T) {
	svc := NewService(newFakeFileRepo(), newFakeStorage(), nil)
	_, err := svc.Upload(context.Background(), UploadCommand{
		UserID:      "user-1",
		Filename:    "test.txt",
		ContentType: "text/plain",
		Size:        maxFileSize + 1,
		Content:     strings.NewReader("hello"),
	})
	if err != ErrTooLarge {
		t.Fatalf("expected ErrTooLarge, got %v", err)
	}
}