package user

import (
	"context"
	"testing"
)

type fakeUserReader struct {
	users map[string]*User
}

func newFakeUserReader() *fakeUserReader {
	return &fakeUserReader{users: map[string]*User{}}
}

func (f *fakeUserReader) FindByID(_ context.Context, id string) (*User, error) {
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, ErrNotFound
}

func (f *fakeUserReader) FindByEmail(_ context.Context, email string) (*User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, ErrNotFound
}

func (f *fakeUserReader) FindByRole(_ context.Context, role Role) ([]*User, error) {
	var out []*User
	for _, u := range f.users {
		if role == "" || u.Role == role {
			out = append(out, u)
		}
	}
	return out, nil
}

type fakeUserWriter struct {
	saved []*User
}

func newFakeUserWriter() *fakeUserWriter {
	return &fakeUserWriter{saved: []*User{}}
}

func (f *fakeUserWriter) Save(_ context.Context, u *User) error {
	f.saved = append(f.saved, u)
	return nil
}

func TestServiceCreate(t *testing.T) {
	reader := newFakeUserReader()
	writer := newFakeUserWriter()

	svc := NewService(reader, writer, nil)

	u, err := svc.Create(context.Background(), CreateUserCommand{
		Email:    "test@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if u.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", u.Email)
	}
	if u.Role != RoleUser {
		t.Errorf("expected role user, got %s", u.Role)
	}
	if len(writer.saved) != 1 {
		t.Errorf("expected 1 saved user, got %d", len(writer.saved))
	}
}

func TestServiceCreateDuplicate(t *testing.T) {
	reader := newFakeUserReader()
	reader.users["1"] = &User{ID: "1", Email: "test@example.com"}
	writer := newFakeUserWriter()

	svc := NewService(reader, writer, nil)

	_, err := svc.Create(context.Background(), CreateUserCommand{
		Email:    "test@example.com",
		Password: "password123",
	})
	if err != ErrEmailTaken {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}
