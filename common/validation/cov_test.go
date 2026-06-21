package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCov_Bag_NilReceiverIsSafe(t *testing.T) {
	t.Parallel()

	var b *Bag
	b.Add(RequiredCode, "field", "msg") // no-op, must not panic
	require.Equal(t, 0, b.Len())
	require.False(t, b.HasAny())
	require.True(t, b.IsEmpty())
	require.Empty(t, b.Failures())
}

func TestCov_Bag_IsEmpty(t *testing.T) {
	t.Parallel()

	b := NewBag()
	require.True(t, b.IsEmpty())

	b.Add(RequiredCode, "field", "msg")
	require.False(t, b.IsEmpty())
}
