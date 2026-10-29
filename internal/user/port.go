package user

import "context"

// UserReader is the read capability for users.
type UserReader interface {
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByRole(ctx context.Context, role Role) ([]*User, error)
}

// UserWriter is the write capability for users.
type UserWriter interface {
	Save(ctx context.Context, u *User) error
}
