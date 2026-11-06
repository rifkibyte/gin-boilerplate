package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rifkibyte/gin-boilerplate/internal/user"
)

type fakeUserFinder struct {
	users map[string]*user.User
}

func newFakeUserFinder() *fakeUserFinder {
	return &fakeUserFinder{users: map[string]*user.User{}}
}

func (f *fakeUserFinder) FindByEmail(_ context.Context, email string) (*user.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, user.ErrNotFound
}

func (f *fakeUserFinder) FindByID(_ context.Context, id string) (*user.User, error) {
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, user.ErrNotFound
}

type fakeUserWriter struct {
	saved []*user.User
}

func newFakeUserWriter() *fakeUserWriter {
	return &fakeUserWriter{saved: []*user.User{}}
}

func (f *fakeUserWriter) Save(_ context.Context, u *user.User) error {
	f.saved = append(f.saved, u)
	return nil
}

type fakeTokenIssuer struct{}

func (fakeTokenIssuer) IssueAccessToken(_ context.Context, userID, role string) (string, error) {
	return "access-" + userID, nil
}
func (fakeTokenIssuer) IssueRefreshToken(_ context.Context, userID string) (string, error) {
	return "refresh-" + userID, nil
}
func (fakeTokenIssuer) VerifyAccessToken(_ context.Context, token string) (string, string, error) {
	return "u1", "user", nil
}
func (fakeTokenIssuer) VerifyRefreshToken(_ context.Context, token string) (string, error) {
	return "u1", nil
}

type fakeTokenStore struct {
	data map[string]string
}

func newFakeTokenStore() *fakeTokenStore {
	return &fakeTokenStore{data: map[string]string{}}
}

func (f *fakeTokenStore) Save(_ context.Context, token, userID string, _ time.Duration) error {
	_ = token
	_ = userID
	return nil
}

func (f *fakeTokenStore) Get(_ context.Context, token string) (string, error) {
	return "u1", nil
}

func (f *fakeTokenStore) Delete(_ context.Context, token string) error {
	return nil
}

type fakeMailer struct{}

func (fakeMailer) Send(_ context.Context, to, subject, body string) error {
	return nil
}

func TestServiceLoginInvalid(t *testing.T) {
	finder := newFakeUserFinder()
	svc := NewService(finder, newFakeUserWriter(), fakeTokenIssuer{}, newFakeTokenStore(), fakeMailer{}, nil)
	_, err := svc.Login(context.Background(), LoginCommand{Email: "x@y.com", Password: "pw"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestServiceRegisterDuplicate(t *testing.T) {
	finder := newFakeUserFinder()
	finder.users["1"] = &user.User{ID: "1", Email: "test@example.com"}
	svc := NewService(finder, newFakeUserWriter(), fakeTokenIssuer{}, newFakeTokenStore(), fakeMailer{}, nil)
	_, err := svc.Register(context.Background(), RegisterCommand{Email: "test@example.com", Password: "password123"})
	if !errors.Is(err, user.ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}
