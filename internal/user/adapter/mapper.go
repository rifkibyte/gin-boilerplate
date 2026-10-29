package adapter

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/rifkibyte/gin-boilerplate/internal/user"
)

// toDomain converts a persistence record to a domain entity.
//
// The conversion can fail because the record holds a uuid.UUID and the domain
// holds a string — parsing is required.
func toDomain(r UserRecord) (*user.User, error) {
	return &user.User{
		ID:        r.ID.String(),
		Email:     r.Email,
		Password:  r.Password,
		Role:      user.Role(r.Role),
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}, nil
}

// toRecord converts a domain entity to a persistence record.
func toRecord(u *user.User) (UserRecord, error) {
	id, err := uuid.Parse(u.ID)
	if err != nil {
		return UserRecord{}, fmt.Errorf("adapter: parsing user id %q: %w", u.ID, err)
	}

	return UserRecord{
		ID:        id,
		Email:     u.Email,
		Password:  u.Password,
		Role:      string(u.Role),
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}, nil
}
