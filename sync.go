package dsa

import (
	"context"
	"sync"
)

// After executes the provided function in a separate goroutine and returns a channel that is closed when the function completes.
// This is useful for running a function asynchronously and waiting for its completion.
func After(fn func()) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		fn()
		close(ch)
	}()
	return ch
}

// ToChannel converts a function that returns a value of type T into a channel that sends that value.
// This is useful for running a function asynchronously and getting its result through a channel.
func ToChannel[T any](fn func() T) <-chan T {
	ch := make(chan T)
	go func() {
		ch <- fn()
		close(ch)
	}()
	return ch
}

// Stopper is a channel that can be used to signal when a task should stop.
// It can be used to implement a simple stop mechanism for long-running tasks.
type Stopper chan struct{}

// NewStopper creates a new Stopper channel.
func NewStopper() Stopper {
	return make(Stopper)
}

// Wait returns a channel that is closed when the Stopper is stopped.
func (s Stopper) Wait() <-chan struct{} {
	return s
}

// Stop sends a signal to the Stopper to indicate that it should stop.
// It closes the channel to prevent further sends.
func (s Stopper) Stop() {
	s <- struct{}{}
	close(s)
}

// Stopped checks if the Stopper has been stopped.
func (s Stopper) Stopped() bool {
	select {
	case _, ok := <-s:
		if !ok {
			return true
		} else {
			return false
		}
	default:
		return false
	}
}

// WorkerPool creates a pool of workers that can process tasks concurrently.
// It takes a context, the size of the pool, a work queue, and a function to process the tasks.
// The function returns two Stopper channels: one for stopping the workers and one for waiting for their completion.
func WorkerPool[W any](ctx context.Context, size int, workQueue Queue[W], doWork func(W)) (stopper Stopper, waiter Stopper) {
	stopper = NewStopper()
	waiter = NewStopper()

	if ctx == nil {
		ctx = context.Background()
	}

	go func() {
		defer waiter.Stop()

		wg := sync.WaitGroup{}
		workers := make(chan W, size)
		for range size {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for work := range workers {
					doWork(work)
				}
			}()
		}

		running := true
		for running {
			select {
			case <-stopper:
				running = false
			case <-ctx.Done():
				running = false
			case w := <-ToChannel(workQueue.Dequeue):
				workers <- w
			}
		}

		close(workers)
		wg.Wait()
	}()

	return
}

// Execute processes tasks from a work queue as fast as work is available.
// A goroutines is started for each task, and the provided function is called with the task as an argument.
// The function returns two Stopper channels: one for stopping the execution and one for waiting for completion.
func Executor[W any](ctx context.Context, workQueue Queue[W], doWork func(W)) (stopper Stopper, waiter Stopper) {
	stopper = NewStopper()
	waiter = NewStopper()

	if ctx == nil {
		ctx = context.Background()
	}

	go func() {
		defer waiter.Stop()

		running := true
		for running {
			select {
			case <-stopper:
				running = false
			case <-ctx.Done():
				running = false
			case w := <-ToChannel(workQueue.Dequeue):
				go doWork(w)
			}
		}
	}()

	return
}
