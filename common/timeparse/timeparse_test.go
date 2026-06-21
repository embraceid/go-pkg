package timeparse_test

import (
	"testing"
	"time"

	"pkg.embrace.id/common/timeparse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStrict(t *testing.T) {
	t.Parallel()

	t.Run("canonical date succeeds", func(t *testing.T) {
		t.Parallel()
		result, ok := timeparse.ParseStrict("2024-03-15", timeparse.LayoutDate)
		require.True(t, ok)
		require.NotNil(t, result)
		assert.Equal(t, 2024, result.Year())
		assert.Equal(t, time.March, result.Month())
		assert.Equal(t, 15, result.Day())
	})

	t.Run("non-canonical date rejected", func(t *testing.T) {
		t.Parallel()
		result, ok := timeparse.ParseStrict("2024-3-5", timeparse.LayoutDate)
		assert.Nil(t, result)
		assert.False(t, ok)
	})

	t.Run("invalid string rejected", func(t *testing.T) {
		t.Parallel()
		result, ok := timeparse.ParseStrict("not-a-date", timeparse.LayoutDate)
		assert.Nil(t, result)
		assert.False(t, ok)
	})
}

func TestParseOptional(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil true", func(t *testing.T) {
		t.Parallel()
		result, ok := timeparse.ParseOptional(nil, timeparse.LayoutDate)
		assert.Nil(t, result)
		assert.True(t, ok)
	})

	t.Run("empty string returns nil true", func(t *testing.T) {
		t.Parallel()
		s := ""
		result, ok := timeparse.ParseOptional(&s, timeparse.LayoutDate)
		assert.Nil(t, result)
		assert.True(t, ok)
	})

	t.Run("valid date returns parsed time true", func(t *testing.T) {
		t.Parallel()
		s := "2024-03-15"
		result, ok := timeparse.ParseOptional(&s, timeparse.LayoutDate)
		require.True(t, ok)
		require.NotNil(t, result)
		assert.Equal(t, 2024, result.Year())
		assert.Equal(t, time.March, result.Month())
		assert.Equal(t, 15, result.Day())
	})

	t.Run("invalid format returns nil false", func(t *testing.T) {
		t.Parallel()
		s := "15-03-2024"
		result, ok := timeparse.ParseOptional(&s, timeparse.LayoutDate)
		assert.Nil(t, result)
		assert.False(t, ok)
	})

	t.Run("valid RFC3339 with custom layout", func(t *testing.T) {
		t.Parallel()
		s := "2024-03-15T10:00:00Z"
		result, ok := timeparse.ParseOptional(&s, time.RFC3339)
		require.True(t, ok)
		require.NotNil(t, result)
		assert.Equal(t, 2024, result.Year())
	})

	t.Run("invalid RFC3339 returns nil false", func(t *testing.T) {
		t.Parallel()
		s := "not-a-time"
		result, ok := timeparse.ParseOptional(&s, time.RFC3339)
		assert.Nil(t, result)
		assert.False(t, ok)
	})
}
