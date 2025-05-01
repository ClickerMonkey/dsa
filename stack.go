package dsa

import (
	"iter"
	"sync"
)

type Stack[T any] interface {
	Push(T)
	Pop() T
	Top() T
	Len() int
	IsEmpty() bool
	Values() iter.Seq[T]
}

type SliceStack[T any] []T

func (s *SliceStack[T]) Push(value T) {
	*s = append(*s, value)
}
func (s *SliceStack[T]) Pop() T {
	if len(*s) == 0 {
		var empty T
		return empty
	}
	last := len(*s) - 1
	popped := (*s)[last]
	*s = (*s)[:last]
	return popped
}
func (s SliceStack[T]) Top() T {
	if len(s) == 0 {
		var empty T
		return empty
	}
	last := len(s) - 1
	top := s[last]
	return top
}
func (s SliceStack[T]) Len() int {
	return len(s)
}
func (s SliceStack[T]) IsEmpty() bool {
	return len(s) == 0
}
func (s SliceStack[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range s {
			if !yield(v) {
				break
			}
		}
	}
}

type WaitStack[T any] struct {
	stack  Stack[T]
	lock   sync.Mutex
	signal sync.Cond
}

var _ Stack[any] = &WaitStack[any]{}

func NewWaitStack[T any](stack Stack[T]) *WaitStack[T] {
	wq := &WaitStack[T]{
		stack: stack,
		lock:  sync.Mutex{},
	}
	wq.signal = *sync.NewCond(&wq.lock)
	return wq
}

func (wq *WaitStack[T]) Push(item T) {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	wq.stack.Push(item)
	wq.signal.Signal()
}

func (wq *WaitStack[T]) Pop() T {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	for wq.stack.Len() == 0 {
		wq.signal.Wait()
	}
	return wq.stack.Pop()
}

func (wq *WaitStack[T]) Top() T {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	if wq.stack.Len() == 0 {
		return zero[T]()
	}
	return wq.stack.Top()
}

func (ts *WaitStack[T]) Len() int {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	return ts.stack.Len()
}

func (ts *WaitStack[T]) IsEmpty() bool {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	return ts.stack.IsEmpty()
}

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
