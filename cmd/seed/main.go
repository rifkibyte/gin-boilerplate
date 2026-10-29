package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rifkibyte/gin-boilerplate/internal/config"
	"github.com/rifkibyte/gin-boilerplate/internal/user"
	useradapter "github.com/rifkibyte/gin-boilerplate/internal/user/adapter"
	"github.com/rifkibyte/gin-boilerplate/pkg/database"
	"github.com/rifkibyte/gin-boilerplate/pkg/id"
	"github.com/rifkibyte/gin-boilerplate/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(logger.Config{
		Level:   cfg.Log.Level,
		Format:  logger.Format(cfg.Log.Format),
		Service: cfg.Log.Service,
	})
	defer func() { _ = log.Sync() }()

	db, err := database.NewPostgres(cfg.DB)
	if err != nil {
		log.Fatal("database connection failed", zap.Error(err))
	}
	defer func() { _ = database.Close(db) }()

	repo := useradapter.NewUserRepositoryGORM(db)

	ctx := context.Background()

	seeds := []struct {
		email    string
		password string
		role     user.Role
	}{
		{"admin@example.com", "admin123", user.RoleAdmin},
		{"user@example.com", "user12345", user.RoleUser},
	}

	for _, s := range seeds {
		existing, err := repo.FindByEmail(ctx, s.email)
		if err != nil && err != user.ErrNotFound {
			log.Fatal("checking seed user", zap.Error(err))
		}
		if existing != nil {
			log.Info("seed user already exists, skipping", zap.String("email", s.email))
			fmt.Printf("seed: %s already exists, nothing to do\n", s.email)
			continue
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(s.password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal("hashing password", zap.Error(err))
		}

		u := &user.User{
			ID:       id.NewV7(),
			Email:    s.email,
			Password: string(hashed),
			Role:     s.role,
		}
		if err := repo.Save(ctx, u); err != nil {
			log.Fatal("saving seed user", zap.Error(err))
		}

		log.Info("seeded user", zap.String("email", u.Email), zap.String("role", string(u.Role)))
		fmt.Printf("seed: %s created (%s / %s)\n", u.Email, s.email, s.password)
	}
}
