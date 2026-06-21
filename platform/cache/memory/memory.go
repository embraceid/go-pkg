package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	sharedcache "pkg.embrace.id/platform/cache"
)

type cacheItem struct {
	value     []byte
	expiresAt time.Time
}

func (i *cacheItem) isExpired() bool {
	return !i.expiresAt.IsZero() && time.Now().After(i.expiresAt)
}

type MemoryCache struct {
	mu      sync.RWMutex
	items   map[string]*cacheItem
	closed  bool
	closeMu sync.RWMutex
}

type MemoryCacheOption func(*MemoryCache)

func NewMemoryCache(opts ...MemoryCacheOption) *MemoryCache {
	c := &MemoryCache{items: make(map[string]*cacheItem)}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	if err := c.validateContextAndState(ctx); err != nil {
		return nil, fmt.Errorf("memory cache get: %w", err)
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists || item.isExpired() {
		return nil, sharedcache.ErrNotFound
	}

	result := make([]byte, len(item.value))
	copy(result, item.value)
	return result, nil
}

func (c *MemoryCache) GetJSON(ctx context.Context, key string, dest any) error {
	if err := c.validateContextAndState(ctx); err != nil {
		return fmt.Errorf("memory cache get json: %w", err)
	}
	if dest == nil {
		return errors.New("memory cache get json: dest cannot be nil")
	}

	data, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("memory cache get json: unmarshal failed: %w", err)
	}
	return nil
}

func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	if err := c.validateContextAndState(ctx); err != nil {
		return false, fmt.Errorf("memory cache exists: %w", err)
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	return exists && !item.isExpired(), nil
}

func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := c.validateContextAndState(ctx); err != nil {
		return fmt.Errorf("memory cache set: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	item := &cacheItem{value: make([]byte, len(value))}
	copy(item.value, value)
	if ttl > 0 {
		item.expiresAt = time.Now().Add(ttl)
	}

	c.items[key] = item
	return nil
}

func (c *MemoryCache) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	if err := c.validateContextAndState(ctx); err != nil {
		return fmt.Errorf("memory cache set json: %w", err)
	}
	if value == nil {
		return sharedcache.ErrNilValue
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("memory cache set json: marshal failed: %w", err)
	}
	return c.Set(ctx, key, data, ttl)
}

func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	if err := c.validateContextAndState(ctx); err != nil {
		return fmt.Errorf("memory cache delete: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
	return nil
}

func (c *MemoryCache) DeleteMulti(ctx context.Context, keys ...string) error {
	if err := c.validateContextAndState(ctx); err != nil {
		return fmt.Errorf("memory cache delete multi: %w", err)
	}
	if len(keys) == 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, key := range keys {
		delete(c.items, key)
	}
	return nil
}

func (c *MemoryCache) Ping(ctx context.Context) error {
	if err := c.validateContextAndState(ctx); err != nil {
		return fmt.Errorf("memory cache ping: %w", err)
	}
	return nil
}

func (c *MemoryCache) Close() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	c.closed = true
	c.mu.Lock()
	c.items = make(map[string]*cacheItem)
	c.mu.Unlock()
	return nil
}

func (c *MemoryCache) validateContextAndState(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is nil")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.closeMu.RLock()
	defer c.closeMu.RUnlock()
	if c.closed {
		return errors.New("cache is closed")
	}
	return nil
}

func (c *MemoryCache) Cleanup() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for key, item := range c.items {
		if item.isExpired() {
			delete(c.items, key)
			count++
		}
	}
	return count
}

func (c *MemoryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}
