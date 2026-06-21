package async

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrPoolClosed       = errors.New("async: pool is closed")
	ErrPoolRunning      = errors.New("async: pool is still running")
	ErrContextCancelled = errors.New("async: context cancelled")
)

type Task func() error

// It collects errors from all tasks and provides them after completion.
type Pool struct {
	mu      sync.Mutex
	tasks   []Task
	errors  []error
	running bool
	closed  bool
	workers int
}

type PoolOption func(*Pool)

// If not set or set to 0, tasks run in separate goroutines.
func WithWorkers(n int) PoolOption {
	return func(p *Pool) {
		p.workers = n
	}
}

func NewPool(opts ...PoolOption) *Pool {
	p := &Pool{
		tasks:   make([]Task, 0),
		errors:  make([]error, 0),
		workers: 0,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Returns ErrPoolClosed if the pool is closed.
func (p *Pool) Add(task Task) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPoolClosed
	}

	p.tasks = append(p.tasks, task)
	return nil
}

// Returns ErrPoolClosed if the pool is closed.
func (p *Pool) AddMany(tasks ...Task) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPoolClosed
	}

	p.tasks = append(p.tasks, tasks...)
	return nil
}

// Size returns the number of queued tasks.
func (p *Pool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.tasks)
}

// If context is cancelled, remaining tasks are skipped.
// Returns all errors collected during execution.
func (p *Pool) Run(ctx context.Context) []error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return []error{ErrPoolClosed}
	}
	if p.running {
		p.mu.Unlock()
		return []error{ErrPoolRunning}
	}

	p.running = true
	tasks := make([]Task, len(p.tasks))
	copy(tasks, p.tasks)
	p.mu.Unlock()
	defer p.markStopped()

	if ctx.Err() != nil {
		return []error{ErrContextCancelled}
	}

	var wg sync.WaitGroup
	var errMu sync.Mutex
	errors := make([]error, 0)

	if p.workers > 0 {
		return p.runWithWorkers(ctx, tasks)
	}

	for _, task := range tasks {
		if ctx.Err() != nil {
			errMu.Lock()
			errors = append(errors, ErrContextCancelled)
			errMu.Unlock()
			break
		}

		wg.Add(1)
		go func(t Task) {
			defer wg.Done()
			if err := t(); err != nil {
				errMu.Lock()
				errors = append(errors, err)
				errMu.Unlock()
			}
		}(task)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		errMu.Lock()
		errors = append(errors, ErrContextCancelled)
		errMu.Unlock()
		<-done
	}

	return errors
}

func (p *Pool) markStopped() {
	p.mu.Lock()
	p.running = false
	p.mu.Unlock()
}

func (p *Pool) runWithWorkers(ctx context.Context, tasks []Task) []error {
	taskCh := make(chan Task, len(tasks))
	var wg sync.WaitGroup
	var errMu sync.Mutex
	errors := make([]error, 0)

	for _, task := range tasks {
		taskCh <- task
	}
	close(taskCh)

	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskCh {
				select {
				case <-ctx.Done():
					errMu.Lock()
					errors = append(errors, ErrContextCancelled)
					errMu.Unlock()
					return
				default:
					if err := task(); err != nil {
						errMu.Lock()
						errors = append(errors, err)
						errMu.Unlock()
					}
					if ctx.Err() != nil {
						errMu.Lock()
						errors = append(errors, ErrContextCancelled)
						errMu.Unlock()
						return
					}
				}
			}
		}()
	}

	wg.Wait()

	return errors
}

// Returns nil if all tasks succeed.
func (p *Pool) RunAndCollect(ctx context.Context) error {
	err := p.Run(ctx)
	if len(err) == 0 {
		return nil
	}
	return err[0]
}

// Returns ErrPoolRunning if the pool is currently running.
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return ErrPoolRunning
	}

	p.closed = true
	return nil
}

// Cannot be called while pool is running.
func (p *Pool) Reset() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return ErrPoolRunning
	}

	p.tasks = make([]Task, 0)
	p.errors = make([]error, 0)
	p.closed = false
	return nil
}

func Parallel(ctx context.Context, tasks ...Task) []error {
	pool := NewPool()
	_ = pool.AddMany(tasks...)
	return pool.Run(ctx)
}

// Returns nil if all tasks succeed.
func ParallelFirstError(ctx context.Context, tasks ...Task) error {
	err := Parallel(ctx, tasks...)
	if len(err) == 0 {
		return nil
	}
	return err[0]
}

// Returns error if all tasks fail.
func FirstSuccess[T any](ctx context.Context, tasks ...func() (T, error)) (T, error) {
	var zero T

	if len(tasks) == 0 {
		return zero, errors.New("async: no tasks provided")
	}

	resultCh := make(chan T, 1)
	errCh := make(chan error, len(tasks))
	done := make(chan struct{})

	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		go func(t func() (T, error)) {
			defer wg.Done()
			result, err := t()
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
			} else {
				select {
				case resultCh <- result:
				default:
				}
			}
		}(task)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return zero, ErrContextCancelled
	case result := <-resultCh:
		return result, nil
	case <-done:
		select {
		case err := <-errCh:
			return zero, err
		default:
			return zero, errors.New("async: all tasks failed with no error")
		}
	}
}
