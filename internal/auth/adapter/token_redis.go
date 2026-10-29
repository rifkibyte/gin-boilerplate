package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/rifkibyte/gin-boilerplate/pkg/cache"
)

// RedisTokenStore stores refresh tokens in Redis with TTL.
type RedisTokenStore struct {
	cache *cache.Client
}

// NewRedisTokenStore constructs a RedisTokenStore.
func NewRedisTokenStore(c *cache.Client) *RedisTokenStore {
	return &RedisTokenStore{cache: c}
}

func (s *RedisTokenStore) Save(ctx context.Context, token, userID string, ttl time.Duration) error {
	return s.cache.Set(ctx, "refresh:"+token, userID, ttl)
}

func (s *RedisTokenStore) Get(ctx context.Context, token string) (string, error) {
	value, err := s.cache.Get(ctx, "refresh:"+token)
	if err != nil {
		return "", fmt.Errorf("token store: get: %w", err)
	}
	if value == "" {
		return "", fmt.Errorf("token store: not found")
	}
	return value, nil
}

func (s *RedisTokenStore) Delete(ctx context.Context, token string) error {
	return s.cache.Delete(ctx, "refresh:"+token)
}