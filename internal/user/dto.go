// Package user — request and response DTOs.
package user

// CreateUserRequest is the payload for POST /users.
//
// Role is deliberately absent: roles are assigned server-side to prevent
// privilege escalation via self-registration.
type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// UserResponse is the public representation of a user.
type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      Role   `json:"role"`
	CreatedAt string `json:"created_at"`
}
