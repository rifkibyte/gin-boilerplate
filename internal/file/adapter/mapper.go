package adapter

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/rifkibyte/gin-boilerplate/internal/file"
)

func toDomain(r FileRecord) (*file.File, error) {
	return &file.File{
		ID:          r.ID.String(),
		UserID:      r.UserID.String(),
		Filename:    r.Filename,
		ContentType: r.ContentType,
		Size:        r.Size,
		Path:        r.Path,
		CreatedAt:   r.CreatedAt,
	}, nil
}

func toRecord(f *file.File) (FileRecord, error) {
	id, err := uuid.Parse(f.ID)
	if err != nil {
		return FileRecord{}, fmt.Errorf("adapter: parsing file id: %w", err)
	}
	userID, err := uuid.Parse(f.UserID)
	if err != nil {
		return FileRecord{}, fmt.Errorf("adapter: parsing user id: %w", err)
	}
	return FileRecord{
		ID:          id,
		UserID:      userID,
		Filename:    f.Filename,
		ContentType: f.ContentType,
		Size:        f.Size,
		Path:        f.Path,
		CreatedAt:   f.CreatedAt,
	}, nil
}