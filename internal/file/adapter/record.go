package adapter

import (
	"time"

	"github.com/google/uuid"
)

// FileRecord is the GORM model for the files table.
type FileRecord struct {
	ID          uuid.UUID `gorm:"primaryKey;type:uuid"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index"`
	Filename    string    `gorm:"not null"`
	ContentType string    `gorm:"not null"`
	Size        int64     `gorm:"not null"`
	Path        string    `gorm:"not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

func (FileRecord) TableName() string {
	return "files"
}
