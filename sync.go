package dsa

import (
	"context"
	"sync"
)

func After(fn func()) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		fn()
		close(ch)
	}()
	return ch
}

func ToChannel[T any](fn func() T) <-chan T {
	ch := make(chan T)
	go func() {
		ch <- fn()
		close(ch)
	}()
	return ch
}

type Stopper chan struct{}

func NewStopper() Stopper {
	return make(Stopper)
}

func (s Stopper) Wait() <-chan struct{} {
	return s
}
func (s Stopper) Stop() {
	s <- struct{}{}
	close(s)
}
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

func WorkerPool[W any](ctx context.Context, size int, workQueue Queue[W], doWork func(*W)) (stopper Stopper, waiter Stopper) {
	stopper = NewStopper()
	waiter = NewStopper()

	if ctx == nil {
		ctx = context.Background()
	}

	go func() {
		defer waiter.Stop()

		wg := sync.WaitGroup{}
		workers := make(chan *W, size)
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

func Executor[W any](ctx context.Context, workQueue Queue[W], doWork func(*W)) (stopper Stopper, waiter Stopper) {
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
