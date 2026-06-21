package cache

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound    = errors.New("cache: key not found")
	ErrNilValue    = errors.New("cache: cannot cache nil value")
	ErrInvalidType = errors.New("cache: invalid type assertion")
)

type CacheReader interface {
	Get(ctx context.Context, key string) ([]byte, error)
	GetJSON(ctx context.Context, key string, dest any) error
	Exists(ctx context.Context, key string) (bool, error)
}

type CacheWriter interface {
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	DeleteMulti(ctx context.Context, keys ...string) error
}

type Cache interface {
	CacheReader
	CacheWriter
	Ping(ctx context.Context) error
	Close() error
}
