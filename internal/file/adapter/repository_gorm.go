package adapter

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/rifkibyte/gin-boilerplate/internal/file"
)

type FileRepositoryGORM struct {
	db *gorm.DB
}

func NewFileRepositoryGORM(db *gorm.DB) *FileRepositoryGORM {
	return &FileRepositoryGORM{db: db}
}

func (r *FileRepositoryGORM) Save(ctx context.Context, f *file.File) error {
	rec, err := toRecord(f)
	if err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		return fmt.Errorf("adapter: creating file: %w", err)
	}
	f.ID = rec.ID.String()
	f.CreatedAt = rec.CreatedAt
	return nil
}

func (r *FileRepositoryGORM) FindByID(ctx context.Context, id string) (*file.File, error) {
	var rec FileRecord
	err := r.db.WithContext(ctx).First(&rec, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, file.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("adapter: finding file: %w", err)
	}
	return toDomain(rec)
}

func (r *FileRepositoryGORM) ListByUser(ctx context.Context, userID string) ([]*file.File, error) {
	var recs []FileRecord
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&recs).Error; err != nil {
		return nil, fmt.Errorf("adapter: listing files: %w", err)
	}
	out := make([]*file.File, 0, len(recs))
	for _, rec := range recs {
		f, err := toDomain(rec)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}