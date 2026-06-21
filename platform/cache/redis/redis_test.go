package redis

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	sharedcache "pkg.embrace.id/platform/cache"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisCache_NewRedisCache(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	cache := NewRedisCache(client)
	assert.NotNil(t, cache)
}

func TestRedisCache_Set_Get(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewRedisCache(client)
	ctx := context.Background()

	tests := []struct {
		name  string
		key   string
		value []byte
		ttl   time.Duration
	}{
		{name: "success with no TTL", key: "test-key-1", value: []byte("test-value-1"), ttl: 0},
		{name: "success with TTL", key: "test-key-2", value: []byte("test-value-2"), ttl: time.Hour},
		{name: "empty value", key: "test-key-3", value: []byte{}, ttl: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cache.Set(ctx, tt.key, tt.value, tt.ttl)
			require.NoError(t, err)

			got, err := cache.Get(ctx, tt.key)
			require.NoError(t, err)
			assert.Equal(t, tt.value, got)
		})
	}
}

func TestRedisCache_Get_NotFound(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewRedisCache(client)
	ctx := context.Background()

	_, err := cache.Get(ctx, "non-existent-key")
	assert.ErrorIs(t, err, sharedcache.ErrNotFound)
}

func TestRedisCache_SetJSON_GetJSON(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewRedisCache(client)
	ctx := context.Background()

	type testStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name  string
		key   string
		value testStruct
		ttl   time.Duration
	}{
		{name: "success", key: "json-key-1", value: testStruct{Name: "test", Value: 42}, ttl: time.Hour},
		{name: "empty struct", key: "json-key-2", value: testStruct{}, ttl: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cache.SetJSON(ctx, tt.key, tt.value, tt.ttl)
			require.NoError(t, err)

			var got testStruct
			err = cache.GetJSON(ctx, tt.key, &got)
			require.NoError(t, err)
			assert.Equal(t, tt.value, got)
		})
	}
}

func TestRedisCache_SetJSON_NilValue(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewRedisCache(client)
	ctx := context.Background()

	err := cache.SetJSON(ctx, "nil-key", nil, time.Hour)
	assert.ErrorIs(t, err, sharedcache.ErrNilValue)
}

func TestRedisCache_GetJSON_NilDest(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewRedisCache(client)
	ctx := context.Background()

	err := cache.SetJSON(ctx, "test-key", map[string]string{"foo": "bar"}, time.Hour)
	require.NoError(t, err)

	err = cache.GetJSON(ctx, "test-key", nil)
	assert.Error(t, err)
}

func TestRedisCache_Exists(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewRedisCache(client)
	ctx := context.Background()

	tests := []struct {
		name     string
		key      string
		setupKey bool
		want     bool
	}{
		{name: "key exists", key: "existing-key", setupKey: true, want: true},
		{name: "key does not exist", key: "non-existent-key", setupKey: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupKey {
				err := cache.Set(ctx, tt.key, []byte("value"), 0)
				require.NoError(t, err)
			}

			got, err := cache.Exists(ctx, tt.key)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRedisCache_Delete(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewRedisCache(client)
	ctx := context.Background()

	err := cache.Set(ctx, "delete-key", []byte("value"), 0)
	require.NoError(t, err)

	exists, err := cache.Exists(ctx, "delete-key")
	require.NoError(t, err)
	assert.True(t, exists)

	err = cache.Delete(ctx, "delete-key")
	require.NoError(t, err)

	exists, err = cache.Exists(ctx, "delete-key")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRedisCache_Delete_NonExistent(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewRedisCache(client)
	ctx := context.Background()

	err := cache.Delete(ctx, "non-existent-key")
	require.NoError(t, err)
}

func TestRedisCache_DeleteMulti(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewRedisCache(client)
	ctx := context.Background()

	keys := []string{"multi-key-1", "multi-key-2", "multi-key-3"}
	for _, key := range keys {
		err := cache.Set(ctx, key, []byte("value"), 0)
		require.NoError(t, err)
	}

	err := cache.DeleteMulti(ctx, keys...)
	require.NoError(t, err)

	for _, key := range keys {
		exists, err := cache.Exists(ctx, key)
		require.NoError(t, err)
		assert.False(t, exists)
	}
}

func TestRedisCache_DeleteMulti_Empty(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewRedisCache(client)
	ctx := context.Background()

	err := cache.DeleteMulti(ctx)
	require.NoError(t, err)
}

func TestRedisCache_Ping(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewRedisCache(client)
	ctx := context.Background()

	err := cache.Ping(ctx)
	require.NoError(t, err)
}

func TestRedisCache_Context_Cancellation(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewRedisCache(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name string
		fn   func() error
	}{
		{name: "Get", fn: func() error { _, err := cache.Get(ctx, "key"); return err }},
		{name: "Set", fn: func() error { return cache.Set(ctx, "key", []byte("value"), 0) }},
		{name: "Delete", fn: func() error { return cache.Delete(ctx, "key") }},
		{name: "Exists", fn: func() error { _, err := cache.Exists(ctx, "key"); return err }},
		{name: "Ping", fn: func() error { return cache.Ping(ctx) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			assert.Error(t, err)
		})
	}
}

func TestRedisCache_Close(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewRedisCache(client)

	err := cache.Close()
	require.NoError(t, err)
}

func TestRedisCache_TTL_Expiration(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewRedisCache(client)
	ctx := context.Background()

	err := cache.Set(ctx, "ttl-key", []byte("value"), 100*time.Millisecond)
	require.NoError(t, err)

	exists, err := cache.Exists(ctx, "ttl-key")
	require.NoError(t, err)
	assert.True(t, exists)

	mr.FastForward(200 * time.Millisecond)

	exists, err = cache.Exists(ctx, "ttl-key")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestValidateContext(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		wantErr bool
	}{
		{name: "valid context", ctx: context.Background(), wantErr: false},
		{name: "nil context", ctx: nil, wantErr: true},
		{name: "cancelled context", ctx: func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateContext(tt.ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func BenchmarkRedisCache_Set(b *testing.B) {
	mr := miniredis.RunT(b)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewRedisCache(client)
	ctx := context.Background()
	value := []byte("benchmark-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.Set(ctx, "bench-key", value, 0)
	}
}

func BenchmarkRedisCache_Get(b *testing.B) {
	mr := miniredis.RunT(b)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewRedisCache(client)
	ctx := context.Background()
	_ = cache.Set(ctx, "bench-key", []byte("benchmark-value"), 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(ctx, "bench-key")
	}
}

func BenchmarkRedisCache_SetJSON(b *testing.B) {
	mr := miniredis.RunT(b)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewRedisCache(client)
	ctx := context.Background()
	value := map[string]any{"name": "test", "value": 42}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.SetJSON(ctx, "bench-key", value, 0)
	}
}

func BenchmarkRedisCache_GetJSON(b *testing.B) {
	mr := miniredis.RunT(b)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewRedisCache(client)
	ctx := context.Background()
	value := map[string]any{"name": "test", "value": 42}
	_ = cache.SetJSON(ctx, "bench-key", value, 0)
	var dest map[string]any

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.GetJSON(ctx, "bench-key", &dest)
	}
}

func init() {
	_, _ = json.Marshal(nil)
	_ = errors.New("")
}
