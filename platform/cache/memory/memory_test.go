package memory

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	sharedcache "pkg.embrace.id/go-pkg/platform/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCache_NewMemoryCache(t *testing.T) {
	cache := NewMemoryCache()
	assert.NotNil(t, cache)
	assert.Equal(t, 0, cache.Size())
}

func TestMemoryCache_Set_Get(t *testing.T) {
	cache := NewMemoryCache()
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
		{name: "binary value", key: "test-key-4", value: []byte{0x00, 0x01, 0x02, 0xFF}, ttl: 0},
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

func TestMemoryCache_Get_NotFound(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	_, err := cache.Get(ctx, "non-existent-key")
	assert.ErrorIs(t, err, sharedcache.ErrNotFound)
}

func TestMemoryCache_Get_Expired(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	err := cache.Set(ctx, "expired-key", []byte("value"), 50*time.Millisecond)
	require.NoError(t, err)

	got, err := cache.Get(ctx, "expired-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("value"), got)

	time.Sleep(100 * time.Millisecond)

	_, err = cache.Get(ctx, "expired-key")
	assert.ErrorIs(t, err, sharedcache.ErrNotFound)
}

func TestMemoryCache_SetJSON_GetJSON(t *testing.T) {
	cache := NewMemoryCache()
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

func TestMemoryCache_SetJSON_NilValue(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	err := cache.SetJSON(ctx, "nil-key", nil, time.Hour)
	assert.ErrorIs(t, err, sharedcache.ErrNilValue)
}

func TestMemoryCache_GetJSON_NilDest(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	err := cache.SetJSON(ctx, "test-key", map[string]string{"foo": "bar"}, time.Hour)
	require.NoError(t, err)

	err = cache.GetJSON(ctx, "test-key", nil)
	assert.Error(t, err)
}

func TestMemoryCache_Exists(t *testing.T) {
	cache := NewMemoryCache()
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

func TestMemoryCache_Exists_Expired(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	err := cache.Set(ctx, "expired-key", []byte("value"), 50*time.Millisecond)
	require.NoError(t, err)

	exists, err := cache.Exists(ctx, "expired-key")
	require.NoError(t, err)
	assert.True(t, exists)

	time.Sleep(100 * time.Millisecond)

	exists, err = cache.Exists(ctx, "expired-key")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestMemoryCache_Delete(t *testing.T) {
	cache := NewMemoryCache()
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

func TestMemoryCache_Delete_NonExistent(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	err := cache.Delete(ctx, "non-existent-key")
	require.NoError(t, err)
}

func TestMemoryCache_DeleteMulti(t *testing.T) {
	cache := NewMemoryCache()
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

func TestMemoryCache_DeleteMulti_Empty(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	err := cache.DeleteMulti(ctx)
	require.NoError(t, err)
}

func TestMemoryCache_Ping(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	err := cache.Ping(ctx)
	require.NoError(t, err)
}

func TestMemoryCache_Close(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	err := cache.Set(ctx, "key", []byte("value"), 0)
	require.NoError(t, err)

	err = cache.Close()
	require.NoError(t, err)

	_, err = cache.Get(ctx, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestMemoryCache_Cleanup(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	err := cache.Set(ctx, "no-expire", []byte("value1"), 0)
	require.NoError(t, err)
	err = cache.Set(ctx, "expired", []byte("value2"), 50*time.Millisecond)
	require.NoError(t, err)
	err = cache.Set(ctx, "not-yet-expired", []byte("value3"), time.Hour)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	removed := cache.Cleanup()
	assert.Equal(t, 1, removed)

	exists, err := cache.Exists(ctx, "no-expire")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = cache.Exists(ctx, "expired")
	require.NoError(t, err)
	assert.False(t, exists)

	exists, err = cache.Exists(ctx, "not-yet-expired")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestMemoryCache_Size(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	assert.Equal(t, 0, cache.Size())
	_ = cache.Set(ctx, "key1", []byte("value"), 0)
	assert.Equal(t, 1, cache.Size())
	_ = cache.Set(ctx, "key2", []byte("value"), 0)
	assert.Equal(t, 2, cache.Size())
	_ = cache.Delete(ctx, "key1")
	assert.Equal(t, 1, cache.Size())
}

func TestMemoryCache_Context_Cancellation(t *testing.T) {
	cache := NewMemoryCache()
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

func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	const numOps = 100
	done := make(chan bool, numOps*3)

	for i := 0; i < numOps; i++ {
		go func(idx int) {
			key := string(rune('a' + idx%26))
			_ = cache.Set(ctx, key, []byte("value"), 0)
			done <- true
		}(i)
	}

	for i := 0; i < numOps; i++ {
		go func(idx int) {
			key := string(rune('a' + idx%26))
			_, _ = cache.Get(ctx, key)
			done <- true
		}(i)
	}

	for i := 0; i < numOps; i++ {
		go func(idx int) {
			key := string(rune('a' + idx%26))
			_ = cache.Delete(ctx, key)
			done <- true
		}(i)
	}

	for i := 0; i < numOps*3; i++ {
		<-done
	}
}

func TestMemoryCache_ValueIsolation(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	original := []byte("original")
	err := cache.Set(ctx, "key", original, 0)
	require.NoError(t, err)

	original[0] = 'X'

	got, err := cache.Get(ctx, "key")
	require.NoError(t, err)
	assert.Equal(t, []byte("original"), got)

	got[0] = 'Y'

	got2, err := cache.Get(ctx, "key")
	require.NoError(t, err)
	assert.Equal(t, []byte("original"), got2)
}

func BenchmarkMemoryCache_Set(b *testing.B) {
	cache := NewMemoryCache()
	ctx := context.Background()
	value := []byte("benchmark-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.Set(ctx, "bench-key", value, 0)
	}
}

func BenchmarkMemoryCache_Get(b *testing.B) {
	cache := NewMemoryCache()
	ctx := context.Background()
	_ = cache.Set(ctx, "bench-key", []byte("benchmark-value"), 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(ctx, "bench-key")
	}
}

func BenchmarkMemoryCache_SetJSON(b *testing.B) {
	cache := NewMemoryCache()
	ctx := context.Background()
	value := map[string]any{"name": "test", "value": 42}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.SetJSON(ctx, "bench-key", value, 0)
	}
}

func BenchmarkMemoryCache_GetJSON(b *testing.B) {
	cache := NewMemoryCache()
	ctx := context.Background()
	value := map[string]any{"name": "test", "value": 42}
	_ = cache.SetJSON(ctx, "bench-key", value, 0)
	var dest map[string]any

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.GetJSON(ctx, "bench-key", &dest)
	}
}

func BenchmarkMemoryCache_Concurrent(b *testing.B) {
	cache := NewMemoryCache()
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := string(rune('a' + i%26))
			if i%2 == 0 {
				_ = cache.Set(ctx, key, []byte("value"), 0)
			} else {
				_, _ = cache.Get(ctx, key)
			}
			i++
		}
	})
}

var _, _ = json.Marshal(nil)
