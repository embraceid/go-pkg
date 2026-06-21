package async

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startRunningPool(t *testing.T, pool *Pool) func() {
	t.Helper()

	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})

	require.NoError(t, pool.Add(func() error {
		close(started)
		<-release
		return nil
	}))

	go func() {
		pool.Run(context.Background())
		close(done)
	}()

	<-started

	return func() {
		close(release)
		<-done
	}
}

func TestPool_New(t *testing.T) {
	pool := NewPool()
	assert.NotNil(t, pool)
	assert.Equal(t, 0, pool.Size())
}

func TestPool_AddOperations(t *testing.T) {
	tests := []struct {
		name   string
		act    func(pool *Pool) error
		assert func(t *testing.T, pool *Pool, err error)
	}{
		{
			name: "add single task",
			act: func(pool *Pool) error {
				return pool.Add(func() error { return nil })
			},
			assert: func(t *testing.T, pool *Pool, err error) {
				require.NoError(t, err)
				assert.Equal(t, 1, pool.Size())
			},
		},
		{
			name: "add many tasks",
			act: func(pool *Pool) error {
				return pool.AddMany(
					func() error { return nil },
					func() error { return nil },
					func() error { return nil },
				)
			},
			assert: func(t *testing.T, pool *Pool, err error) {
				require.NoError(t, err)
				assert.Equal(t, 3, pool.Size())
			},
		},
		{
			name: "add on closed pool",
			act: func(pool *Pool) error {
				require.NoError(t, pool.Close())
				return pool.Add(func() error { return nil })
			},
			assert: func(t *testing.T, pool *Pool, err error) {
				assert.ErrorIs(t, err, ErrPoolClosed)
				assert.Equal(t, 0, pool.Size())
			},
		},
		{
			name: "add many on closed pool",
			act: func(pool *Pool) error {
				require.NoError(t, pool.Close())
				return pool.AddMany(func() error { return nil }, func() error { return nil })
			},
			assert: func(t *testing.T, pool *Pool, err error) {
				assert.ErrorIs(t, err, ErrPoolClosed)
				assert.Equal(t, 0, pool.Size())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewPool()
			err := tt.act(pool)
			tt.assert(t, pool, err)
		})
	}
}

func TestPool_Run(t *testing.T) {
	testErr := errors.New("test error")

	tests := []struct {
		name    string
		newPool func() *Pool
		setup   func(t *testing.T, pool *Pool, counter *int32) (context.Context, func())
		assert  func(t *testing.T, errs []error, counter *int32)
	}{
		{
			name:    "success",
			newPool: func() *Pool { return NewPool() },
			setup: func(t *testing.T, pool *Pool, counter *int32) (context.Context, func()) {
				for i := 0; i < 10; i++ {
					require.NoError(t, pool.Add(func() error {
						atomic.AddInt32(counter, 1)
						return nil
					}))
				}
				return context.Background(), func() {}
			},
			assert: func(t *testing.T, errs []error, counter *int32) {
				assert.Empty(t, errs)
				assert.Equal(t, int32(10), atomic.LoadInt32(counter))
			},
		},
		{
			name:    "with task errors",
			newPool: func() *Pool { return NewPool() },
			setup: func(t *testing.T, pool *Pool, _ *int32) (context.Context, func()) {
				require.NoError(t, pool.AddMany(
					func() error { return nil },
					func() error { return testErr },
					func() error { return nil },
					func() error { return testErr },
				))
				return context.Background(), func() {}
			},
			assert: func(t *testing.T, errs []error, _ *int32) {
				require.Len(t, errs, 2)
				for _, err := range errs {
					assert.ErrorIs(t, err, testErr)
				}
			},
		},
		{
			name:    "context cancellation",
			newPool: func() *Pool { return NewPool() },
			setup: func(t *testing.T, pool *Pool, _ *int32) (context.Context, func()) {
				for i := 0; i < 10; i++ {
					require.NoError(t, pool.Add(func() error {
						time.Sleep(100 * time.Millisecond)
						return nil
					}))
				}
				ctx, cancel := context.WithCancel(context.Background())
				timer := time.AfterFunc(10*time.Millisecond, cancel)
				return ctx, func() {
					timer.Stop()
					cancel()
				}
			},
			assert: func(t *testing.T, errs []error, _ *int32) {
				assert.Contains(t, errs, ErrContextCancelled)
			},
		},
		{
			name:    "already running",
			newPool: func() *Pool { return NewPool() },
			setup: func(t *testing.T, pool *Pool, _ *int32) (context.Context, func()) {
				cleanup := startRunningPool(t, pool)
				return context.Background(), cleanup
			},
			assert: func(t *testing.T, errs []error, _ *int32) {
				require.Len(t, errs, 1)
				assert.ErrorIs(t, errs[0], ErrPoolRunning)
			},
		},
		{
			name:    "with workers",
			newPool: func() *Pool { return NewPool(WithWorkers(3)) },
			setup: func(t *testing.T, pool *Pool, counter *int32) (context.Context, func()) {
				for i := 0; i < 10; i++ {
					require.NoError(t, pool.Add(func() error {
						atomic.AddInt32(counter, 1)
						return nil
					}))
				}
				return context.Background(), func() {}
			},
			assert: func(t *testing.T, errs []error, counter *int32) {
				assert.Empty(t, errs)
				assert.Equal(t, int32(10), atomic.LoadInt32(counter))
			},
		},
		{
			name:    "context already cancelled",
			newPool: func() *Pool { return NewPool() },
			setup: func(t *testing.T, pool *Pool, _ *int32) (context.Context, func()) {
				require.NoError(t, pool.Add(func() error { return nil }))
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			assert: func(t *testing.T, errs []error, _ *int32) {
				assert.Contains(t, errs, ErrContextCancelled)
			},
		},
		{
			name:    "closed pool",
			newPool: func() *Pool { return NewPool() },
			setup: func(t *testing.T, pool *Pool, _ *int32) (context.Context, func()) {
				require.NoError(t, pool.Close())
				return context.Background(), func() {}
			},
			assert: func(t *testing.T, errs []error, _ *int32) {
				assert.Contains(t, errs, ErrPoolClosed)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := tt.newPool()
			var counter int32
			ctx, cleanup := tt.setup(t, pool, &counter)
			defer cleanup()

			err := pool.Run(ctx)
			tt.assert(t, err, &counter)
		})
	}
}

func TestPool_RunWithAlreadyCancelledContextCanReset(t *testing.T) {
	pool := NewPool()
	require.NoError(t, pool.Add(func() error { return nil }))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := pool.Run(ctx)

	require.Len(t, err, 1)
	assert.ErrorIs(t, err[0], ErrContextCancelled)
	assert.NoError(t, pool.Reset())
}

func TestPool_RunWaitsForStartedTasksAfterContextCancellation(t *testing.T) {
	pool := NewPool()
	started := make(chan struct{})
	release := make(chan struct{})
	runDone := make(chan []error, 1)

	require.NoError(t, pool.Add(func() error {
		close(started)
		<-release
		return nil
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		runDone <- pool.Run(ctx)
	}()

	<-started
	cancel()

	select {
	case errs := <-runDone:
		t.Fatalf("Run returned before started task finished: %v", errs)
	case <-time.After(20 * time.Millisecond):
	}

	close(release)

	select {
	case errs := <-runDone:
		assert.Contains(t, errs, ErrContextCancelled)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Run did not return after started task finished")
	}
}

func TestPool_RunWithWorkersReportsCancellationWhileTaskIsActive(t *testing.T) {
	pool := NewPool(WithWorkers(1))
	started := make(chan struct{})
	release := make(chan struct{})
	runDone := make(chan []error, 1)

	require.NoError(t, pool.Add(func() error {
		close(started)
		<-release
		return nil
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		runDone <- pool.Run(ctx)
	}()

	<-started
	cancel()
	close(release)

	select {
	case errs := <-runDone:
		assert.Contains(t, errs, ErrContextCancelled)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Run did not return after worker task finished")
	}
}

func TestPool_RunAndCollect(t *testing.T) {
	tests := []struct {
		name    string
		tasks   []Task
		wantErr bool
	}{
		{
			name: "all success",
			tasks: []Task{
				func() error { return nil },
				func() error { return nil },
			},
			wantErr: false,
		},
		{
			name: "with error",
			tasks: []Task{
				func() error { return errors.New("error") },
				func() error { return nil },
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewPool()
			require.NoError(t, pool.AddMany(tt.tasks...))

			err := pool.RunAndCollect(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPool_StateTransitions(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(t *testing.T, pool *Pool) func()
		act    func(pool *Pool) error
		assert func(t *testing.T, pool *Pool, err error)
	}{
		{
			name: "close idle pool",
			setup: func(t *testing.T, pool *Pool) func() {
				return func() {}
			},
			act: func(pool *Pool) error {
				return pool.Close()
			},
			assert: func(t *testing.T, pool *Pool, err error) {
				require.NoError(t, err)
				assert.ErrorIs(t, pool.Add(func() error { return nil }), ErrPoolClosed)
			},
		},
		{
			name: "close while running",
			setup: func(t *testing.T, pool *Pool) func() {
				return startRunningPool(t, pool)
			},
			act: func(pool *Pool) error {
				return pool.Close()
			},
			assert: func(t *testing.T, _ *Pool, err error) {
				assert.ErrorIs(t, err, ErrPoolRunning)
			},
		},
		{
			name: "reset completed pool",
			setup: func(t *testing.T, pool *Pool) func() {
				require.NoError(t, pool.Add(func() error { return nil }))
				require.NoError(t, pool.Add(func() error { return nil }))
				assert.Equal(t, 2, pool.Size())
				assert.Empty(t, pool.Run(context.Background()))
				return func() {}
			},
			act: func(pool *Pool) error {
				return pool.Reset()
			},
			assert: func(t *testing.T, pool *Pool, err error) {
				require.NoError(t, err)
				assert.Equal(t, 0, pool.Size())
			},
		},
		{
			name: "reset while running",
			setup: func(t *testing.T, pool *Pool) func() {
				return startRunningPool(t, pool)
			},
			act: func(pool *Pool) error {
				return pool.Reset()
			},
			assert: func(t *testing.T, _ *Pool, err error) {
				assert.ErrorIs(t, err, ErrPoolRunning)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewPool()
			cleanup := tt.setup(t, pool)
			defer cleanup()

			err := tt.act(pool)
			tt.assert(t, pool, err)
		})
	}
}

func TestParallel(t *testing.T) {
	var counter int32
	tasks := []Task{
		func() error { atomic.AddInt32(&counter, 1); return nil },
		func() error { atomic.AddInt32(&counter, 1); return nil },
		func() error { atomic.AddInt32(&counter, 1); return nil },
	}

	err := Parallel(context.Background(), tasks...)
	assert.Empty(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&counter))
}

func TestParallelFirstError(t *testing.T) {
	tests := []struct {
		name    string
		tasks   []Task
		wantErr bool
	}{
		{
			name: "all success",
			tasks: []Task{
				func() error { return nil },
				func() error { return nil },
			},
			wantErr: false,
		},
		{
			name: "with error",
			tasks: []Task{
				func() error { return errors.New("error") },
				func() error { return nil },
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParallelFirstError(context.Background(), tt.tasks...)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFirstSuccess(t *testing.T) {
	tests := []struct {
		name    string
		tasks   []func() (string, error)
		want    string
		wantErr bool
	}{
		{
			name: "first succeeds",
			tasks: []func() (string, error){
				func() (string, error) { return "success", nil },
				func() (string, error) { return "", errors.New("error") },
			},
			want:    "success",
			wantErr: false,
		},
		{
			name: "all fail",
			tasks: []func() (string, error){
				func() (string, error) { return "", errors.New("error1") },
				func() (string, error) { return "", errors.New("error2") },
			},
			want:    "",
			wantErr: true,
		},
		{
			name:    "no tasks",
			tasks:   []func() (string, error){},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FirstSuccess(context.Background(), tt.tasks...)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

// Benchmarks

func BenchmarkPool_Run(b *testing.B) {
	pool := NewPool()
	task := func() error { return nil }

	for i := 0; i < 100; i++ {
		_ = pool.Add(task)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pool.Run(context.Background())
		_ = pool.Reset()
		for j := 0; j < 100; j++ {
			_ = pool.Add(task)
		}
	}
}

func BenchmarkPool_RunWithWorkers(b *testing.B) {
	task := func() error { return nil }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool := NewPool(WithWorkers(10))
		for j := 0; j < 100; j++ {
			_ = pool.Add(task)
		}
		_ = pool.Run(context.Background())
	}
}

func BenchmarkParallel(b *testing.B) {
	tasks := make([]Task, 100)
	for i := range tasks {
		tasks[i] = func() error { return nil }
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Parallel(context.Background(), tasks...)
	}
}
