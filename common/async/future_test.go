package async

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type intFutureCallback struct {
	result int
	err    error
}

func awaitIntFutureCallbacks(t *testing.T, ch <-chan intFutureCallback, count int) []intFutureCallback {
	t.Helper()

	callbacks := make([]intFutureCallback, 0, count)
	for len(callbacks) < count {
		select {
		case callback := <-ch:
			callbacks = append(callbacks, callback)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("callback was not called")
		}
	}

	return callbacks
}

func TestFuture_NewFuture(t *testing.T) {
	f := NewFuture[int]()
	assert.NotNil(t, f)
	assert.False(t, f.IsDone())
}

func TestFuture_Get(t *testing.T) {
	testErr := errors.New("test error")

	tests := []struct {
		name    string
		setup   func(t *testing.T, f *Future[int]) (context.Context, func())
		want    int
		wantErr error
	}{
		{
			name: "returns value after successful set",
			setup: func(t *testing.T, f *Future[int]) (context.Context, func()) {
				assert.True(t, f.Set(42, nil))
				return context.Background(), func() {}
			},
			want: 42,
		},
		{
			name: "returns stored error",
			setup: func(t *testing.T, f *Future[int]) (context.Context, func()) {
				assert.True(t, f.Set(0, testErr))
				return context.Background(), func() {}
			},
			want:    0,
			wantErr: testErr,
		},
		{
			name: "returns cancellation error when context is cancelled",
			setup: func(t *testing.T, f *Future[int]) (context.Context, func()) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			want:    0,
			wantErr: ErrFutureCancelled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFuture[int]()
			ctx, cleanup := tt.setup(t, f)
			defer cleanup()

			result, err := f.Get(ctx)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestFuture_SetOnlyOnce(t *testing.T) {
	f := NewFuture[int]()

	firstSet := f.Set(1, nil)
	secondSet := f.Set(2, nil)

	assert.True(t, firstSet)
	assert.False(t, secondSet)

	result, err := f.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, result)
}

func TestFuture_GetWithTimeout(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(f *Future[int])
		timeout time.Duration
		want    int
		wantErr error
	}{
		{
			name:    "completes before timeout",
			setup:   func(f *Future[int]) { go func() { time.Sleep(10 * time.Millisecond); f.Set(42, nil) }() },
			timeout: 100 * time.Millisecond,
			want:    42,
			wantErr: nil,
		},
		{
			name:    "times out",
			setup:   func(f *Future[int]) {},
			timeout: 10 * time.Millisecond,
			want:    0,
			wantErr: ErrFutureCancelled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFuture[int]()
			tt.setup(f)

			result, err := f.GetWithTimeout(tt.timeout)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestFuture_GetOr(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(f *Future[int])
		defaultVal int
		want       int
	}{
		{
			name:       "returns result on success",
			setup:      func(f *Future[int]) { f.Set(42, nil) },
			defaultVal: 0,
			want:       42,
		},
		{
			name:       "returns default on error",
			setup:      func(f *Future[int]) { f.Set(0, errors.New("error")) },
			defaultVal: 99,
			want:       99,
		},
		{
			name:       "returns default on cancellation",
			setup:      func(f *Future[int]) {},
			defaultVal: 99,
			want:       99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFuture[int]()
			tt.setup(f)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()

			result := f.GetOr(ctx, tt.defaultVal)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestFuture_OnComplete(t *testing.T) {
	tests := []struct {
		name             string
		callbackCount    int
		completeBeforeOn bool
		result           int
		err              error
	}{
		{
			name:          "calls callback when future completes later",
			callbackCount: 1,
			result:        42,
		},
		{
			name:             "calls callback when future already completed",
			callbackCount:    1,
			completeBeforeOn: true,
			result:           42,
		},
		{
			name:          "calls every registered callback",
			callbackCount: 5,
			result:        42,
		},
		{
			name:          "passes stored error to callbacks",
			callbackCount: 2,
			result:        0,
			err:           errors.New("callback error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFuture[int]()
			callbackCh := make(chan intFutureCallback, tt.callbackCount)

			if tt.completeBeforeOn {
				assert.True(t, f.Set(tt.result, tt.err))
			}

			for i := 0; i < tt.callbackCount; i++ {
				f.OnComplete(func(result int, err error) {
					callbackCh <- intFutureCallback{result: result, err: err}
				})
			}

			if !tt.completeBeforeOn {
				assert.True(t, f.Set(tt.result, tt.err))
			}

			callbacks := awaitIntFutureCallbacks(t, callbackCh, tt.callbackCount)
			for _, callback := range callbacks {
				assert.Equal(t, tt.result, callback.result)
				if tt.err != nil {
					assert.ErrorIs(t, callback.err, tt.err)
				} else {
					assert.NoError(t, callback.err)
				}
			}
		})
	}
}

func TestAsyncConstructors(t *testing.T) {
	asyncErr := errors.New("async error")

	tests := []struct {
		name    string
		build   func() *Future[string]
		want    string
		wantErr error
	}{
		{
			name: "async returns result",
			build: func() *Future[string] {
				return Async(func() (string, error) { return "async result", nil })
			},
			want: "async result",
		},
		{
			name: "async returns error",
			build: func() *Future[string] {
				return Async(func() (string, error) { return "", asyncErr })
			},
			wantErr: asyncErr,
		},
		{
			name: "async with context returns result",
			build: func() *Future[string] {
				return AsyncWithContext(context.Background(), func(ctx context.Context) (string, error) {
					return "context result", nil
				})
			},
			want: "context result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.build().Get(context.Background())
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestMap(t *testing.T) {
	mapErr := errors.New("map error")
	sourceErr := errors.New("original error")

	tests := []struct {
		name    string
		source  *Future[int]
		mapFn   func(int) (string, error)
		want    string
		wantErr error
	}{
		{
			name:   "maps successful result",
			source: Resolve(42),
			mapFn: func(x int) (string, error) {
				return "mapped", nil
			},
			want: "mapped",
		},
		{
			name:   "returns mapper error",
			source: Resolve(42),
			mapFn: func(x int) (string, error) {
				return "", mapErr
			},
			wantErr: mapErr,
		},
		{
			name:   "propagates source error",
			source: Reject[int](sourceErr),
			mapFn: func(x int) (string, error) {
				return "should not be called", nil
			},
			wantErr: sourceErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapped := Map(tt.source, tt.mapFn)
			result, err := mapped.Get(context.Background())
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestAll(t *testing.T) {
	allErr := errors.New("second failed")

	tests := []struct {
		name    string
		build   func() []*Future[string]
		want    []string
		wantErr error
	}{
		{
			name: "returns all results",
			build: func() []*Future[string] {
				return []*Future[string]{Resolve("first"), Resolve("second"), Resolve("third")}
			},
			want: []string{"first", "second", "third"},
		},
		{
			name: "returns first error",
			build: func() []*Future[string] {
				return []*Future[string]{Resolve("first"), Reject[string](allErr), Resolve("third")}
			},
			wantErr: allErr,
		},
		{
			name: "returns empty slice for no futures",
			build: func() []*Future[string] {
				return nil
			},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := All(context.Background(), tt.build()...)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, results)
			}
		})
	}
}

func TestAny(t *testing.T) {
	tests := []struct {
		name    string
		build   func() []*Future[string]
		want    string
		wantErr bool
	}{
		{
			name: "returns first successful future",
			build: func() []*Future[string] {
				return []*Future[string]{
					Async(func() (string, error) {
						time.Sleep(10 * time.Millisecond)
						return "first", nil
					}),
					Async(func() (string, error) {
						time.Sleep(100 * time.Millisecond)
						return "second", nil
					}),
				}
			},
			want: "first",
		},
		{
			name: "returns error when all fail",
			build: func() []*Future[string] {
				return []*Future[string]{Reject[string](errors.New("error 1")), Reject[string](errors.New("error 2"))}
			},
			wantErr: true,
		},
		{
			name: "returns error with no futures",
			build: func() []*Future[string] {
				return nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Any(context.Background(), tt.build()...)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestRace(t *testing.T) {
	f1 := Async(func() (string, error) {
		time.Sleep(10 * time.Millisecond)
		return "", errors.New("first error")
	})

	f2 := Async(func() (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "second", nil
	})

	_, err := Race(context.Background(), f1, f2)
	assert.Error(t, err)
}

func TestCompletedFutures(t *testing.T) {
	rejectErr := errors.New("rejected")

	tests := []struct {
		name     string
		build    func() *Future[int]
		want     int
		wantErr  error
		wantDone bool
	}{
		{
			name: "completed future returns value",
			build: func() *Future[int] {
				return CompletedFuture(42, nil)
			},
			want:     42,
			wantDone: true,
		},
		{
			name: "resolve returns value",
			build: func() *Future[int] {
				return Resolve(42)
			},
			want:     42,
			wantDone: true,
		},
		{
			name: "reject returns error",
			build: func() *Future[int] {
				return Reject[int](rejectErr)
			},
			want:     0,
			wantErr:  rejectErr,
			wantDone: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.build()
			assert.Equal(t, tt.wantDone, f.IsDone())

			result, err := f.Get(context.Background())
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestPromise_New(t *testing.T) {
	p := NewPromise[int]()
	assert.NotNil(t, p)
	assert.NotNil(t, p.Future())
	assert.False(t, p.IsDone())
}

func TestPromise_Completion(t *testing.T) {
	rejectErr := errors.New("rejected")

	tests := []struct {
		name     string
		complete func(p *Promise[string]) bool
		want     string
		wantErr  error
		wantDone bool
		wantSet  bool
	}{
		{
			name: "resolve stores value",
			complete: func(p *Promise[string]) bool {
				return p.Resolve("resolved")
			},
			want:     "resolved",
			wantDone: true,
			wantSet:  true,
		},
		{
			name: "reject stores error",
			complete: func(p *Promise[string]) bool {
				return p.Reject(rejectErr)
			},
			wantErr:  rejectErr,
			wantDone: true,
			wantSet:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPromise[string]()
			assert.Equal(t, tt.wantSet, tt.complete(p))
			assert.Equal(t, tt.wantDone, p.IsDone())

			result, err := p.Future().Get(context.Background())
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestPromise_OnlyOnce(t *testing.T) {
	p := NewPromise[int]()

	first := p.Resolve(1)
	second := p.Resolve(2)

	assert.True(t, first)
	assert.False(t, second)

	result, err := p.Future().Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, result)
}

// Benchmarks

func BenchmarkFuture_Get(b *testing.B) {
	f := NewFuture[int]()
	f.Set(42, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.Get(context.Background())
	}
}

func BenchmarkFuture_Set(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f := NewFuture[int]()
		f.Set(42, nil)
	}
}

func BenchmarkAsync(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f := Async(func() (int, error) {
			return 42, nil
		})
		_, _ = f.Get(context.Background())
	}
}

func BenchmarkAll(b *testing.B) {
	futures := make([]*Future[int], 10)
	for i := range futures {
		futures[i] = Resolve(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = All(context.Background(), futures...)
	}
}

func BenchmarkPromise_Resolve(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := NewPromise[int]()
		p.Resolve(42)
	}
}
