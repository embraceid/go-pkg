package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	sharedcache "pkg.embrace.id/platform/cache"
	"github.com/stretchr/testify/require"
)

// newFastClient returns a client that fails fast (no retries, short timeouts) so
// the "server is gone" error-path tests don't sit through backoff retries.
func newFastClient(t *testing.T) (*RedisCache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr:         mr.Addr(),
		MaxRetries:   -1,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})
	return NewRedisCache(client), mr
}

func covCancelledCtx() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestCov_Redis_NewAppliesOptions(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	called := false
	c := NewRedisCache(client, func(*RedisCache) { called = true })
	require.NotNil(t, c)
	require.True(t, called)
}

func TestCov_Redis_JSONErrors(t *testing.T) {
	c, _ := newFastClient(t)
	ctx := context.Background()
	var dst struct{ A int }

	// validateContext error branches
	require.Error(t, c.GetJSON(covCancelledCtx(), "k", &dst))
	require.Error(t, c.SetJSON(covCancelledCtx(), "k", map[string]int{"a": 1}, 0))
	require.Error(t, c.DeleteMulti(covCancelledCtx(), "a", "b"))

	// missing key → underlying Get error propagates
	require.ErrorIs(t, c.GetJSON(ctx, "missing", &dst), sharedcache.ErrNotFound)

	// stored bytes are not valid JSON → unmarshal error
	require.NoError(t, c.Set(ctx, "raw", []byte("not-json"), 0))
	require.Error(t, c.GetJSON(ctx, "raw", &dst))

	// channels cannot be JSON-encoded → marshal error
	require.Error(t, c.SetJSON(ctx, "k", make(chan int), 0))
}

func TestCov_Redis_ClientErrorsWhenServerGone(t *testing.T) {
	c, mr := newFastClient(t)
	ctx := context.Background()
	mr.Close() // operations now fail with a connection error (not redis.Nil)

	_, err := c.Get(ctx, "k")
	require.Error(t, err)
	_, err = c.Exists(ctx, "k")
	require.Error(t, err)
	require.Error(t, c.Set(ctx, "k", []byte("v"), 0))
	require.Error(t, c.Delete(ctx, "k"))
	require.Error(t, c.DeleteMulti(ctx, "a", "b"))
	require.Error(t, c.Ping(ctx))
}

func TestCov_Redis_CloseTwiceErrors(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	c := NewRedisCache(client)
	require.NoError(t, c.Close())
	require.Error(t, c.Close()) // second close → client already closed
}
