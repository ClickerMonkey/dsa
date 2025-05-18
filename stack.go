package dsa

import (
	"iter"
	"sync"
)

// Stack is a generic stack interface that defines the basic operations of a stack.
// It is a LIFO (Last In First Out) data structure that allows adding and removing elements
// in a specific order.
type Stack[T any] interface {
	// Push adds a new element to the top of the stack.
	Push(T)
	// Pop removes and returns the top element of the stack.
	// If the stack is empty, it returns the zero value of T.
	Pop() T
	// Top returns the top element of the stack without removing it.
	// If the stack is empty, it returns the zero value of T.
	Top() T
	// Len returns the number of elements in the stack.
	// This may be O(n) for some implementations.
	Len() int
	// IsEmpty returns true if the stack is empty.
	// This is expected to be O(1) for most implementations.
	IsEmpty() bool
	// Values returns an iterator of the values in the stack.
	Values() iter.Seq[T]
}

// StackDrainEmpty drains the stack into a channel.
// The channel is closed when the stack is empty.
func StackDrainEmpty[T any](s Stack[T], len int) <-chan T {
	return DrainEmpty(s.IsEmpty, s.Pop, len)
}

// StackDrainDone creates a channel that will receive values from the queue.
// It's assumed the queue has a blocking Dequeue method. Only the given
// done channel is used to cancel the channel, otherwise the spawned goroutine will
// run forever waiting for values to be added to the queue.
func StackDrainDone[T any](done <-chan bool, q Stack[T], len int) <-chan T {
	return DrainDone(done, q.Pop, len)
}

// SliceStack is a stack implementation using a slice.
type SliceStack[T any] []T

// Push adds a new element to the top of the stack.
func (s *SliceStack[T]) Push(value T) {
	*s = append(*s, value)
}

// Pop removes and returns the top element of the stack.
// If the stack is empty, it returns the zero value of T.
func (s *SliceStack[T]) Pop() T {
	if len(*s) == 0 {
		return Zero[T]()
	}
	last := len(*s) - 1
	popped := (*s)[last]
	*s = (*s)[:last]
	return popped
}

// Top returns the top element of the stack without removing it.
// If the stack is empty, it returns the zero value of T.
func (s SliceStack[T]) Top() T {
	if len(s) == 0 {
		return Zero[T]()
	}
	last := len(s) - 1
	top := s[last]
	return top
}

// Len returns the number of elements in the stack.
func (s SliceStack[T]) Len() int {
	return len(s)
}

// IsEmpty returns true if the stack is empty.
func (s SliceStack[T]) IsEmpty() bool {
	return len(s) == 0
}

// Values returns an iterator of the values in the stack.
func (s SliceStack[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range s {
			if !yield(v) {
				break
			}
		}
	}
}

// WaitStack is a thread-safe stack implementation that wraps another stack.
// The Pop method blocks until an item is available to pop.
type WaitStack[T any] struct {
	stack  Stack[T]
	lock   sync.Mutex
	signal sync.Cond
}

var _ Stack[any] = &WaitStack[any]{}

// NewWaitStack creates a new WaitStack that wraps the given stack.
func NewWaitStack[T any](stack Stack[T]) *WaitStack[T] {
	wq := &WaitStack[T]{
		stack: stack,
		lock:  sync.Mutex{},
	}
	wq.signal = *sync.NewCond(&wq.lock)
	return wq
}

// Push adds a new item to the stack and signals any waiting
func (wq *WaitStack[T]) Push(item T) {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	wq.stack.Push(item)
	wq.signal.Signal()
}

// Pop removes and returns the top item from the stack.
// If the stack is empty, it blocks until an item is available.
func (wq *WaitStack[T]) Pop() T {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	for wq.stack.IsEmpty() {
		wq.signal.Wait()
	}
	return wq.stack.Pop()
}

// Peek returns the top item from the stack without removing it.
// If the stack is empty, it returns the zero value of T.
func (wq *WaitStack[T]) Top() T {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	if wq.stack.IsEmpty() {
		return Zero[T]()
	}
	return wq.stack.Top()
}

// Len returns the number of items in the stack.
func (ts *WaitStack[T]) Len() int {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	return ts.stack.Len()
}

// IsEmpty returns true if the stack is empty.
func (ts *WaitStack[T]) IsEmpty() bool {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	return ts.stack.IsEmpty()
}

// Values returns an iterator of the values in the stack.
// It locks the stack while iterating, so it is not safe to modify the stack
// while iterating over it.
func (ts *WaitStack[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		ts.lock.Lock()
		defer ts.lock.Unlock()
		for v := range ts.stack.Values() {
			if !yield(v) {
				break
			}
		}
	}
}

// A thread-safe stack implementation.
type SyncStack[T any] struct {
	stack Stack[T]
	lock  sync.Mutex
}

var _ Stack[any] = &SyncStack[any]{}

// NewSyncStack creates a new SyncStack that wraps the given stack.
func NewSyncStack[T any](stack Stack[T]) *SyncStack[T] {
	return &SyncStack[T]{stack: stack}
}

// Push adds a new item to the stack.
// It is safe to call this method concurrently.
func (s *SyncStack[T]) Push(item T) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.stack.Push(item)
}

// Pop removes and returns the top item from the stack.
// If the stack is empty, a zero value of T is returned.
// It is safe to call this method concurrently.
func (s *SyncStack[T]) Pop() T {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.stack.Pop()
}

// Top returns the top item from the stack without removing it.
// If the stack is empty, a zero value of T is returned.
// It is safe to call this method concurrently.
func (s *SyncStack[T]) Top() T {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.stack.Top()
}

// Len returns the number of items in the stack.
// It is safe to call this method concurrently.
func (s *SyncStack[T]) Len() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.stack.Len()
}

// IsEmpty returns true if the stack is empty.
// It is safe to call this method concurrently.
func (s *SyncStack[T]) IsEmpty() bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.stack.IsEmpty()
}

// Values returns an iterator of the values in the stack.
// The order of the values matches the order of the internal stack.
// It locks the stack while iterating, so it is not safe to modify the stack
// while iterating over it.
func (s *SyncStack[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		s.lock.Lock()
		defer s.lock.Unlock()
		for v := range s.stack.Values() {
			if !yield(v) {
				break
			}
		}
	}
}
