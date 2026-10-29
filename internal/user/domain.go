// Package user is the inner (domain) package for the user feature.
//
// It imports nothing from gin, gorm, or net/http. The linter enforces this.
package user

import "time"

// Role is a user's authorization role.
type Role string

const (
	// RoleAdmin can manage other users.
	RoleAdmin Role = "admin"
	// RoleUser is the default role for self-registered users.
	RoleUser Role = "user"
)

// User is the domain entity.
type User struct {
	ID        string
	Email     string
	Password  string // already hashed
	Role      Role
	CreatedAt time.Time
	UpdatedAt time.Time
}
