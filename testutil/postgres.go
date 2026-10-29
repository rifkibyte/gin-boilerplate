//go:build integration

// Package testutil provides helpers for integration tests using Testcontainers.
package testutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer wraps a Testcontainers PostgreSQL instance.
type PostgresContainer struct {
	Container *postgres.PostgresContainer
	DSN       string
}

// NewPostgresContainer starts a PostgreSQL container for testing.
func NewPostgresContainer(ctx context.Context, t *testing.T) *PostgresContainer {
	t.Helper()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*1e9),
		),
	)
	if err != nil {
		t.Fatalf("starting postgres container: %v", err)
	}

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("getting connection string: %v", err)
	}

	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("terminating postgres container: %v", err)
		}
	})

	return &PostgresContainer{
		Container: pgContainer,
		DSN:       dsn,
	}
}

// FormatDSN is a helper for tests that need a DSN string.
func FormatDSN(host string, port int, user, password, dbname string) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)
}
