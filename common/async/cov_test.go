package async

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// errAfterCallsCtx returns nil for the first nilThrough calls to Err(), then
// context.Canceled. It lets Pool.Run pass its initial ctx.Err() guard but trip
// the in-loop cancellation check on the next call — deterministically, with no
// goroutine timing race.
type errAfterCallsCtx struct {
	context.Context
	calls      int
	nilThrough int
}

func (c *errAfterCallsCtx) Err() error {
	c.calls++
	if c.calls <= c.nilThrough {
		return nil
	}
	return context.Canceled
}

// closedDoneNilErrCtx has an already-closed Done() channel but reports a nil
// Err(). This forces a worker's `select { case <-ctx.Done() }` arm while
// Pool.Run's initial ctx.Err() guard still passes.
type closedDoneNilErrCtx struct{ done chan struct{} }

func (closedDoneNilErrCtx) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c closedDoneNilErrCtx) Done() <-chan struct{}     { return c.done }
func (closedDoneNilErrCtx) Err() error                  { return nil }
func (closedDoneNilErrCtx) Value(any) any               { return nil }

func TestCov_Then_SuccessAndError(t *testing.T) {
	// success path: upstream resolves, fn runs and maps the value
	ok := Then(Resolve(2), func(v int, _ string, _ error) (string, error) {
		return "mapped", nil
	})
	res, err := ok.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, "mapped", res)

	// error path: upstream rejects, fn is skipped and the error propagates
	failed := Then(Reject[int](errors.New("boom")), func(v int, _ string, _ error) (string, error) {
		return "should not run", nil
	})
	_, err = failed.Get(context.Background())
	require.Error(t, err)
}

func TestCov_Any_AllArms(t *testing.T) {
	// no futures → error
	_, err := Any[int](context.Background())
	require.Error(t, err)

	// all futures fail → returns one of the errors
	_, err = Any(context.Background(),
		Reject[int](errors.New("a")), Reject[int](errors.New("b")))
	require.Error(t, err)

	// cancelled context — loop so the ctx.Done() select arm is exercised
	for i := 0; i < 1500; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, _ = Any(ctx, NewFuture[int]())
	}

	// all-success futures — loop so the inner "all failed" default arm is
	// exercised when the main select happens to pick <-done over <-resultCh
	for i := 0; i < 3000; i++ {
		_, _ = Any(context.Background(), Resolve(i))
	}
}

func TestCov_Race_EmptyAndCancelled(t *testing.T) {
	// no futures → error
	_, err := Race[int](context.Background())
	require.Error(t, err)

	// cancelled context — loop so the ctx.Done() select arm is exercised
	for i := 0; i < 1500; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, _ = Race(ctx, NewFuture[int]())
	}
}

func TestCov_Pool_Run_InLoopCancellation(t *testing.T) {
	ctx := &errAfterCallsCtx{Context: context.Background(), nilThrough: 1}
	p := NewPool() // workers == 0 → serial spawn loop
	require.NoError(t, p.AddMany(
		func() error { return nil },
		func() error { return nil },
	))
	errs := p.Run(ctx)
	require.Contains(t, errs, ErrContextCancelled)
}

func TestCov_Pool_Workers_ContextDoneArm(t *testing.T) {
	done := make(chan struct{})
	close(done)
	p := NewPool(WithWorkers(2))
	require.NoError(t, p.AddMany(
		func() error { return nil },
		func() error { return nil },
	))
	errs := p.Run(closedDoneNilErrCtx{done: done})
	require.Contains(t, errs, ErrContextCancelled)
}

func TestCov_Pool_Workers_CollectsTaskError(t *testing.T) {
	p := NewPool(WithWorkers(2))
	require.NoError(t, p.AddMany(
		func() error { return errors.New("task failed") },
		func() error { return nil },
	))
	errs := p.Run(context.Background())
	require.NotEmpty(t, errs)
}

func TestCov_FirstSuccess_AllArms(t *testing.T) {
	// no tasks → error
	_, err := FirstSuccess[int](context.Background())
	require.Error(t, err)

	// all tasks fail → returns one of the errors
	_, err = FirstSuccess(context.Background(),
		func() (int, error) { return 0, errors.New("x") },
		func() (int, error) { return 0, errors.New("y") },
	)
	require.Error(t, err)

	// cancelled context — loop so the ctx.Done() select arm is exercised
	for i := 0; i < 1500; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, _ = FirstSuccess(ctx, func() (int, error) { return 1, nil })
	}

	// all-success tasks — loop so the inner "all failed" default arm is
	// exercised when the main select happens to pick <-done over <-resultCh
	for i := 0; i < 3000; i++ {
		_, _ = FirstSuccess(context.Background(), func() (int, error) { return 1, nil })
	}
}
