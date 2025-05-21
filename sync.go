package dsa

import (
	"context"
	"sync"
	"time"
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

// Waits for the Stopper to be closed
func (s Stopper) Wait() {
	if !s.Stopped() {
		<-s
	}
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

// WithCancel creates a new context with a cancel function that is triggered when the Stopper is stopped.
// It returns the new context and the Stopper itself.
func (s Stopper) WithCancel(ctx context.Context) (context.Context, Stopper) {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-s:
			cancel()
		case <-ctx.Done():
			s.Stop()
		}
	}()

	return ctx, s
}

// GoWaiter runs a function in a separate goroutine and returns a Stopper that can be used to wait for its completion.
func GoWaiter(fn func()) Stopper {
	waiter := NewStopper()
	go func() {
		fn()
		waiter.Stop()
	}()
	return waiter
}

// RacePair returns a channel that sends values from either of the two input channels.
// It will close when it receives a value from either channel. If both channels close or
// are closed and return no values, the returned channel will also close without a value.
func RacePair[T any](a <-chan T, b <-chan T) <-chan T {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}

	ch := make(chan T)
	go func() {
		select {
		case v, ok := <-a:
			if ok {
				ch <- v
			} else if v, ok := <-b; ok {
				ch <- v
			}
		case v, ok := <-b:
			if ok {
				ch <- v
			} else if v, ok := <-a; ok {
				ch <- v
			}
		}
		close(ch)
	}()
	return ch
}

// WorkerPool creates a pool of workers that can process tasks concurrently.
// It takes a context, the size of the pool, a work queue, and a function to process the tasks.
// The function returns two Stopper channels: one for stopping the workers and one for waiting for their completion.
func WorkerPool[W any](done <-chan struct{}, size int, workQueue <-chan W, doWork func(W)) (stopper Stopper, waiter Stopper) {
	stopper = NewStopper()
	waiter = NewStopper()

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
			case <-done:
				running = false
			case w, ok := <-workQueue:
				if ok {
					workers <- w
				} else {
					running = false
				}
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
func Executor[W any](done <-chan struct{}, workQueue <-chan W, doWork func(W)) (stopper Stopper, waiter Stopper) {
	stopper = NewStopper()
	waiter = NewStopper()

	go func() {
		defer waiter.Stop()

		running := true
		for running {
			select {
			case <-stopper:
				running = false
			case <-done:
				running = false
			case w, ok := <-workQueue:
				if ok {
					go doWork(w)
				} else {
					running = false
				}
			}
		}
	}()

	return
}

// FrequencyLoop is a structure that allows running a function at a specified interval.
// It can be used to create a loop that runs a task periodically, with a minimum interval between executions.
// A timeout can also be set for a child context passed to the doWork function to not allow the interval to
// exceed a certain duration. The loop will stop when the Stopper is stopped or when the context is done.
type FrequencyLoop struct {
	// A maximum amount of time to allow the doWork function to run.
	// If the doWork function takes longer than this time, the context will be cancelled.
	Timeout time.Duration
	// The ideal interval at which to run the doWork function.
	// The doWork function will be called at this interval, but it may not be exact.
	// The actual interval may be longer if the doWork function takes longer to run.
	Interval time.Duration
	// The minimum interval between executions of the doWork function.
	// This is used to prevent the doWork function from being called too frequently.
	MinInterval time.Duration
	// A stopper that can be used to stop the loop.
	Stopper Stopper
}

// Sync runs the doWork function at the specified interval.
// This is a blocking call and will not return until the Stopper is stopped or the context is done.
// The doWork function will be called with a context that is cancelled if the Timeout is exceeded.
func (fw *FrequencyLoop) Sync(ctx context.Context, doWork func(ctx context.Context)) {
	if ctx == nil {
		ctx = context.Background()
	}

	if fw.Stopper == nil {
		fw.Stopper = NewStopper()
	}

	lastRun := time.Now()
	for !fw.Stopper.Stopped() {
		fw.do(ctx, doWork)

		endRun := time.Now()
		elapsed := endRun.Sub(lastRun)
		lastRun = endRun

		waitTime := max(fw.Interval-elapsed, fw.MinInterval)

		select {
		case <-fw.Stopper:
			return
		case <-ctx.Done():
			return
		case <-time.After(waitTime):
			// Continue to the next iteration
		}
	}
}

func (fw *FrequencyLoop) do(ctx context.Context, doWork func(ctx context.Context)) {
	doCtx := ctx
	if fw.Timeout > 0 {
		var cancel func()
		doCtx, cancel = context.WithTimeout(ctx, fw.Timeout)
		defer cancel()
	}
	doWork(doCtx)
}

// DrainEmpty drains a function into a channel until the isEmpty function returns true.
// The channel is closed when the isEmpty function returns true.
func DrainEmpty[T any](isEmpty func() bool, take func() T, len int) <-chan T {
	ch := make(chan T, len)
	go func() {
		for !isEmpty() {
			ch <- take()
		}
		close(ch)
	}()
	return ch
}

// DrainDone drains a function into a channel until the done channel is closed.
// The channel is closed when the done channel is closed.
func DrainDone[T any](done <-chan struct{}, take func() T, len int) <-chan T {
	ch := make(chan T, len)
	go func() {
		for {
			select {
			case <-done:
				close(ch)
				return
			case v := <-ToChannel(take):
				ch <- v
			}
		}
	}()
	return ch
}
