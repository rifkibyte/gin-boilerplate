package adapter

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/rifkibyte/gin-boilerplate/internal/user"
)

// UserRepositoryGORM implements user.UserReader and user.UserWriter using GORM.
type UserRepositoryGORM struct {
	db *gorm.DB
}

// NewUserRepositoryGORM constructs a UserRepositoryGORM.
func NewUserRepositoryGORM(db *gorm.DB) *UserRepositoryGORM {
	return &UserRepositoryGORM{db: db}
}

// FindByID looks up a user by primary key.
func (r *UserRepositoryGORM) FindByID(ctx context.Context, id string) (*user.User, error) {
	var rec UserRecord
	err := r.db.WithContext(ctx).First(&rec, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("adapter: finding user by id: %w", err)
	}

	return toDomain(rec)
}

// FindByEmail looks up a user by email.
func (r *UserRepositoryGORM) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	var rec UserRecord
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("adapter: finding user by email: %w", err)
	}

	return toDomain(rec)
}

// FindByRole lists users with the given role. An empty role returns all users.
func (r *UserRepositoryGORM) FindByRole(ctx context.Context, role user.Role) ([]*user.User, error) {
	var recs []UserRecord
	query := r.db.WithContext(ctx)
	if role != "" {
		query = query.Where("role = ?", string(role))
	}
	if err := query.Find(&recs).Error; err != nil {
		return nil, fmt.Errorf("adapter: finding users by role: %w", err)
	}

	users := make([]*user.User, 0, len(recs))
	for _, rec := range recs {
		u, err := toDomain(rec)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// Save inserts a new user.
func (r *UserRepositoryGORM) Save(ctx context.Context, u *user.User) error {
	rec, err := toRecord(u)
	if err != nil {
		return err
	}

	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		return fmt.Errorf("adapter: creating user: %w", err)
	}

	u.ID = rec.ID.String()
	u.CreatedAt = rec.CreatedAt
	u.UpdatedAt = rec.UpdatedAt
	return nil
}
