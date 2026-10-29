//go:build integration

// Package testutil provides helpers for integration tests using Testcontainers.
package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// resolveMigrationsDir finds the migrations directory on disk.
func resolveMigrationsDir(path string) (string, error) {
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		return "", fmt.Errorf("migrations dir not found: %s", path)
	}

	candidates := []string{
		"migrations",
		"../migrations",
		"../../migrations",
		"../../../migrations",
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			abs, err := filepath.Abs(c)
			if err == nil {
				return abs, nil
			}
			return c, nil
		}
	}

	return "", fmt.Errorf("migrations directory not found")
}

// RunMigrations applies all up migrations to the database identified by dsn.
//
// It is safe to call repeatedly. A cleanup is registered with t to roll the
// schema back down when the test finishes.
func RunMigrations(t *testing.T, dsn string, migrationsPath ...string) error {
	t.Helper()

	path := ""
	if len(migrationsPath) > 0 {
		path = migrationsPath[0]
	}

	dir, err := resolveMigrationsDir(path)
	if err != nil {
		return err
	}

	m, err := migrate.New("file://"+dir, dsn)
	if err != nil {
		return fmt.Errorf("creating migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("running migrations: %w", err)
	}

	t.Cleanup(func() {
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			t.Logf("rolling back migrations: %v", err)
		}
		_, _ = m.Close()
	})

	return nil
}

// MigrateUp is an alias for RunMigrations.
func MigrateUp(t *testing.T, dsn string, migrationsPath ...string) error {
	return RunMigrations(t, dsn, migrationsPath...)
}

// ApplyMigrations is an alias for RunMigrations.
func ApplyMigrations(t *testing.T, dsn string, migrationsPath ...string) error {
	return RunMigrations(t, dsn, migrationsPath...)
}

// Migrate is an alias for RunMigrations.
func Migrate(t *testing.T, dsn string, migrationsPath ...string) error {
	return RunMigrations(t, dsn, migrationsPath...)
}
