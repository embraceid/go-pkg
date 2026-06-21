package logger

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCov_GetLogger_ReinitializesWhenNil exercises the lazy-init branch that the
// package-level init() normally makes unreachable. Not parallel: it mutates the
// package singleton.
func TestCov_GetLogger_ReinitializesWhenNil(t *testing.T) {
	saved := logger
	t.Cleanup(func() { logger = saved })

	logger = nil
	got := GetLogger()
	require.NotNil(t, got)
	require.Same(t, got, logger)
}
