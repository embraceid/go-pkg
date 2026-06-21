package memory

import (
	"context"
	"testing"

	sharedcache "pkg.embrace.id/platform/cache"

	"github.com/stretchr/testify/require"
)

func covCancelledCtx() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestCov_Memory_NewAppliesOptions(t *testing.T) {
	called := false
	c := NewMemoryCache(func(*MemoryCache) { called = true })
	require.NotNil(t, c)
	require.True(t, called)
}

func TestCov_Memory_NilContextRejected(t *testing.T) {
	c := NewMemoryCache()
	_, err := c.Get(nil, "k") //nolint:staticcheck // intentionally testing the nil-context guard
	require.Error(t, err)
}

func TestCov_Memory_GetJSON_Errors(t *testing.T) {
	c := NewMemoryCache()
	ctx := context.Background()
	var dst struct{ A int }

	// validateContextAndState error branch
	require.Error(t, c.GetJSON(covCancelledCtx(), "k", &dst))

	// nil destination
	require.Error(t, c.GetJSON(ctx, "k", nil))

	// missing key → underlying Get error propagates
	require.ErrorIs(t, c.GetJSON(ctx, "missing", &dst), sharedcache.ErrNotFound)

	// stored bytes are not valid JSON → unmarshal error
	require.NoError(t, c.Set(ctx, "raw", []byte("not-json"), 0))
	require.Error(t, c.GetJSON(ctx, "raw", &dst))
}

func TestCov_Memory_SetJSON_Errors(t *testing.T) {
	c := NewMemoryCache()
	ctx := context.Background()

	// validateContextAndState error branch
	require.Error(t, c.SetJSON(covCancelledCtx(), "k", map[string]int{"a": 1}, 0))

	// nil value
	require.ErrorIs(t, c.SetJSON(ctx, "k", nil, 0), sharedcache.ErrNilValue)

	// channels cannot be JSON-encoded → marshal error
	require.Error(t, c.SetJSON(ctx, "k", make(chan int), 0))
}

func TestCov_Memory_DeleteMulti_CancelledContext(t *testing.T) {
	c := NewMemoryCache()
	require.Error(t, c.DeleteMulti(covCancelledCtx(), "a", "b"))
}
