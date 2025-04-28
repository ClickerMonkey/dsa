package dsa

import (
	"container/heap"
	"sync"
	"time"
)

type Queue[T any] interface {
	Enqueue(*T)
	Dequeue() *T
	Peek() *T
	Len() int
}

type PriorityQueue[T any] struct {
	items []*T
	less  Less[*T]
}

var _ Queue[any] = &PriorityQueue[any]{}

func NewPriorityQueue[T any](less Less[*T]) *PriorityQueue[T] {
	return &PriorityQueue[T]{
		items: make([]*T, 0),
		less:  less,
	}
}

func (pq PriorityQueue[T]) Len() int { return len(pq.items) }

func (pq PriorityQueue[T]) Less(i, j int) bool {
	return pq.less(pq.items[i], pq.items[j])
}

func (pq PriorityQueue[T]) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
}

func (pq *PriorityQueue[T]) Push(x any) {
	pq.items = append(pq.items, x.(*T))
}

func (pq *PriorityQueue[T]) Pop() any {
	old := pq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	pq.items = old[0 : n-1]
	return item
}

func (pq *PriorityQueue[T]) Enqueue(h *T) {
	heap.Push(pq, h)
}

func (pq *PriorityQueue[T]) Dequeue() *T {
	if len(pq.items) == 0 {
		return nil
	}
	return heap.Pop(pq).(*T)
}

func (pq *PriorityQueue[T]) Peek() *T {
	if len(pq.items) == 0 {
		return nil
	}
	return pq.items[0]
}

type WaitQueue[T any] struct {
	queue  Queue[T]
	lock   sync.Mutex
	signal sync.Cond
}

var _ Queue[any] = &WaitQueue[any]{}

func NewWaitQueue[T any](queue Queue[T]) *WaitQueue[T] {
	wq := &WaitQueue[T]{
		queue: queue,
		lock:  sync.Mutex{},
	}
	wq.signal = *sync.NewCond(&wq.lock)
	return wq
}

func (wq *WaitQueue[T]) Enqueue(item *T) {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	wq.queue.Enqueue(item)
	wq.signal.Signal()
}

func (wq *WaitQueue[T]) Dequeue() *T {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	for wq.queue.Len() == 0 {
		wq.signal.Wait()
	}
	return wq.queue.Dequeue()
}

func (wq *WaitQueue[T]) Peek() *T {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	if wq.queue.Len() == 0 {
		return nil
	}
	return wq.queue.Peek()
}

func (ts *WaitQueue[T]) Len() int {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	return ts.queue.Len()
}

type ReadyQueue[T any] struct {
	priority       *PriorityQueue[T]
	wait           *WaitQueue[T]
	readyState     func() *T
	checkFrequency time.Duration
}

var _ Queue[any] = &ReadyQueue[any]{}

func NewReadyQueue[T any](less Less[*T], readyState func() *T, checkFrequency time.Duration) *ReadyQueue[T] {
	pq := NewPriorityQueue(less)
	wq := NewWaitQueue(pq)

	return &ReadyQueue[T]{
		priority:       pq,
		wait:           wq,
		readyState:     readyState,
		checkFrequency: checkFrequency,
	}
}

func (rq *ReadyQueue[T]) Enqueue(item *T) {
	rq.wait.Enqueue(item)
}
func (rq *ReadyQueue[T]) Dequeue() *T {
	rq.wait.lock.Lock()
	defer rq.wait.lock.Unlock()

	for rq.priority.Len() == 0 {
		rq.wait.signal.Wait()
	}

	for rq.priority.Len() > 0 && !rq.priority.less(rq.priority.Peek(), rq.readyState()) {
		rq.wait.lock.Unlock()
		time.Sleep(rq.checkFrequency)
		rq.wait.lock.Lock()
	}

	return rq.priority.Dequeue()
}

func (rq *ReadyQueue[T]) Peek() *T {
	return rq.wait.Peek()
}

func (rq *ReadyQueue[T]) Len() int {
	return rq.wait.Len()
}
