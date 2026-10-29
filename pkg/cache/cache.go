// Package cache wraps the Redis client used for caching and rate limiting.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps a *redis.Client with convenience methods.
type Client struct {
	rdb *redis.Client
}

// New creates a new Redis client.
func New(addr, password string, db int) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &Client{rdb: rdb}
}

// Ping checks the Redis connection.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("cache: ping: %w", err)
	}

	return nil
}

// Get retrieves a value by key. Returns ("", nil) when the key does not exist.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	value, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}

		return "", fmt.Errorf("cache: get %q: %w", key, err)
	}

	return value, nil
}

// Set stores a value with the given TTL.
func (c *Client) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := c.rdb.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("cache: set %q: %w", key, err)
	}

	return nil
}

// Delete removes a key.
func (c *Client) Delete(ctx context.Context, key string) error {
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("cache: delete %q: %w", key, err)
	}

	return nil
}

// Increment atomically increments a counter and returns the new value.
func (c *Client) Increment(ctx context.Context, key string) (int64, error) {
	value, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("cache: increment %q: %w", key, err)
	}

	return value, nil
}

// Expire sets a TTL on an existing key.
func (c *Client) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if err := c.rdb.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("cache: expire %q: %w", key, err)
	}

	return nil
}

// Close releases the Redis connection.
func (c *Client) Close() error {
	return c.rdb.Close()
}

// Underlying returns the raw *redis.Client for use in adapters that need it.
func (c *Client) Underlying() *redis.Client {
	return c.rdb
}
