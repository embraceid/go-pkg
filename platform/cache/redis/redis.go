package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sharedcache "pkg.embrace.id/platform/cache"
	goredis "github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *goredis.Client
}

type RedisCacheOption func(*RedisCache)

func NewRedisCache(client *goredis.Client, opts ...RedisCacheOption) *RedisCache {
	c := &RedisCache{client: client}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	if err := validateContext(ctx); err != nil {
		return nil, fmt.Errorf("cache get: %w", err)
	}

	result, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, sharedcache.ErrNotFound
		}
		return nil, fmt.Errorf("cache get: %w", err)
	}

	return result, nil
}

func (c *RedisCache) GetJSON(ctx context.Context, key string, dest any) error {
	if err := validateContext(ctx); err != nil {
		return fmt.Errorf("cache get json: %w", err)
	}
	if dest == nil {
		return fmt.Errorf("cache get json: dest cannot be nil")
	}

	data, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("cache get json: unmarshal failed: %w", err)
	}
	return nil
}

func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	if err := validateContext(ctx); err != nil {
		return false, fmt.Errorf("cache exists: %w", err)
	}

	count, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("cache exists: %w", err)
	}
	return count > 0, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := validateContext(ctx); err != nil {
		return fmt.Errorf("cache set: %w", err)
	}
	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("cache set: %w", err)
	}
	return nil
}

func (c *RedisCache) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	if err := validateContext(ctx); err != nil {
		return fmt.Errorf("cache set json: %w", err)
	}
	if value == nil {
		return sharedcache.ErrNilValue
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache set json: marshal failed: %w", err)
	}
	return c.Set(ctx, key, data, ttl)
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	if err := validateContext(ctx); err != nil {
		return fmt.Errorf("cache delete: %w", err)
	}
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("cache delete: %w", err)
	}
	return nil
}

func (c *RedisCache) DeleteMulti(ctx context.Context, keys ...string) error {
	if err := validateContext(ctx); err != nil {
		return fmt.Errorf("cache delete multi: %w", err)
	}
	if len(keys) == 0 {
		return nil
	}
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("cache delete multi: %w", err)
	}
	return nil
}

func (c *RedisCache) Ping(ctx context.Context) error {
	if err := validateContext(ctx); err != nil {
		return fmt.Errorf("cache ping: %w", err)
	}
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("cache ping: %w", err)
	}
	return nil
}

func (c *RedisCache) Close() error {
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("cache close: %w", err)
	}
	return nil
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is nil")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
