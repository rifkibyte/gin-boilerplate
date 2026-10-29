package adapter

import (
	"time"

	"github.com/google/uuid"
)

// UserRecord is the GORM model for the users table.
type UserRecord struct {
	ID        uuid.UUID `gorm:"primaryKey;type:uuid"`
	Email     string    `gorm:"uniqueIndex;not null"`
	Password  string    `gorm:"not null"`
	Role      string    `gorm:"not null;index"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName returns the table name.
func (UserRecord) TableName() string {
	return "users"
}
