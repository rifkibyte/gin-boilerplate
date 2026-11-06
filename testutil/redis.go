//go:build integration

// Package testutil provides helpers for integration tests using Testcontainers.
package testutil

import (
	"context"
	"strings"
	"testing"

	redis "github.com/testcontainers/testcontainers-go/modules/redis"
)

// RedisContainer wraps a Testcontainers Redis instance.
type RedisContainer struct {
	Container *redis.RedisContainer
	Addr      string
}

// NewRedisContainer starts a Redis container for testing.
func NewRedisContainer(ctx context.Context, t *testing.T) *RedisContainer {
	t.Helper()

	rContainer, err := redis.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Fatalf("starting redis container: %v", err)
	}

	connStr, err := rContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("getting redis connection string: %v", err)
	}

	addr := strings.TrimPrefix(connStr, "redis://")
	addr = strings.TrimPrefix(addr, "rediss://")

	t.Cleanup(func() {
		if err := rContainer.Terminate(ctx); err != nil {
			t.Logf("terminating redis container: %v", err)
		}
	})

	return &RedisContainer{
		Container: rContainer,
		Addr:      addr,
	}
}
