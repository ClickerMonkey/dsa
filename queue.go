package dsa

import (
	"container/heap"
	"iter"
	"sync"
	"time"
)

// A queue interface that can be used to enqueue and dequeue values.
// It is a FIFO (First In First Out) data structure that allows adding and removing elements
// in a specific order.
type Queue[T any] interface {
	// Enqueue adds a new element to the end of the queue.
	Enqueue(T)
	// Dequeue removes and returns the first element of the queue.
	// If the queue is empty, it returns the zero value of T.
	Dequeue() T
	// Peek returns the first element of the queue without removing it.
	// If the queue is empty, it returns the zero value of T.
	Peek() T
	// Len returns the number of elements in the queue.
	Len() int
	// IsEmpty returns true if the queue is empty.
	IsEmpty() bool
	// Values returns an iterator of the values in the queue.
	// The order of the values is not guaranteed.
	Values() iter.Seq[T]
}

// A priority queue implementation that uses a heap to store the items.
// This is not a thread-safe queue.
type PriorityQueue[T any] struct {
	items    []T
	less     Less[T]
	setIndex func(T, int)
}

var _ Queue[any] = &PriorityQueue[any]{}
var _ heap.Interface = &PriorityQueue[any]{}

// NewPriorityQueue creates a new priority queue with the given less function.
// The less function orders the items in the queue, the first item in the queue
// is the one that is less than all the others.
func NewPriorityQueue[T any](less Less[T]) *PriorityQueue[T] {
	return &PriorityQueue[T]{
		items:    make([]T, 0),
		less:     less,
		setIndex: func(item T, index int) {},
	}
}

// SetIndex sets the function that will be used to set the index of the items in the queue.
func (pq *PriorityQueue[T]) SetIndex(setIndex func(T, int)) {
	if setIndex == nil {
		pq.setIndex = func(item T, index int) {}
	} else {
		pq.setIndex = setIndex
	}
}

// SetLess sets the less function that will be used to order the items in the queue.
// This will cause the queue to be re-ordered.
func (pq *PriorityQueue[T]) SetLess(less Less[T]) {
	pq.less = less
	pq.Refresh()
}

// GetLess returns the less function that is used to order the items in the queue.
func (pq *PriorityQueue[T]) GetLess() Less[T] {
	return pq.less
}

// Clear clears the queue.
func (pq *PriorityQueue[T]) Clear() {
	pq.items = pq.items[:0]
}

// Len returns the number of items in the queue.
func (pq PriorityQueue[T]) Len() int {
	return len(pq.items)
}

// IsEmpty returns true if the queue is empty.
func (pq PriorityQueue[T]) IsEmpty() bool {
	return len(pq.items) == 0
}

// Less returns true if the item at index i is less than the item at index j.
// This is used by the heap interface to order the items in the queue.
func (pq PriorityQueue[T]) Less(i, j int) bool {
	return pq.less(pq.items[i], pq.items[j])
}

// Swap swaps the items at index i and j.
// This is used by the heap interface to order the items in the queue.
func (pq PriorityQueue[T]) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.setIndex(pq.items[i], i)
	pq.setIndex(pq.items[j], j)
}

// Push adds an item to the queue.
// This is used by the heap interface to add items to the queue.
func (pq *PriorityQueue[T]) Push(x any) {
	item := x.(T)
	pq.setIndex(item, len(pq.items))
	pq.items = append(pq.items, item)
}

// Pop removes the item at the end of the queue.
// This is used by the heap interface to remove items from the queue.
func (pq *PriorityQueue[T]) Pop() any {
	old := pq.items
	n := len(old)
	item := old[n-1]
	pq.setIndex(item, -1)
	old[n-1] = Zero[T]()
	pq.items = old[0 : n-1]
	return item
}

// Enqueue adds an item to the queue.
func (pq *PriorityQueue[T]) Enqueue(h T) {
	heap.Push(pq, h)
}

// Dequeue removes the item at the front of the queue.
// If the queue is empty, it returns the zero value of T.
func (pq *PriorityQueue[T]) Dequeue() T {
	if len(pq.items) == 0 {
		return Zero[T]()
	}
	return heap.Pop(pq).(T)
}

// Peek returns the item at the front of the queue without removing it.
// If the queue is empty, it returns the zero value of T.
func (pq *PriorityQueue[T]) Peek() T {
	if len(pq.items) == 0 {
		return Zero[T]()
	}
	return pq.items[0]
}

// Remove tries to remove the value at the index from the queue.
// True is returned if the value is removed.
func (pq *PriorityQueue[T]) Remove(i int) bool {
	if i >= 0 && i < len(pq.items) {
		heap.Remove(pq, i)
		return true
	}
	return false
}

// Update tells the queue that a value has been changed in a way that would
// affect its order in the queue. True is returned if the value was updated.
func (pq *PriorityQueue[T]) Update(i int) bool {
	if i < 0 || i >= len(pq.items) {
		heap.Fix(pq, i)
		return true
	}
	return false
}

// Refresh updates the order of the items in the priority queue if they've
// become out of order.
func (pq *PriorityQueue[T]) Refresh() {
	heap.Init(pq)
}

// Values returns a sequence of the values in the queue.
// The order of the values is guaranteed to be in the order of least to
// greatest according to the less function.
func (pq *PriorityQueue[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := range HeapIterate(pq) {
			if !yield(pq.items[i]) {
				return
			}
		}
	}
}

// A queue implementation that uses an internal queue where
// the Dequeue method blocks until an item is available.
type WaitQueue[T any] struct {
	queue  Queue[T]
	lock   sync.Mutex
	signal sync.Cond
}

var _ Queue[any] = &WaitQueue[any]{}

// NewWaitQueue creates a new WaitQueue with the given queue.
func NewWaitQueue[T any](queue Queue[T]) *WaitQueue[T] {
	wq := &WaitQueue[T]{
		queue: queue,
		lock:  sync.Mutex{},
	}
	wq.signal = *sync.NewCond(&wq.lock)
	return wq
}

// Enqueue adds an item to the queue and signals any waiting
// goroutines that an item is available.
func (wq *WaitQueue[T]) Enqueue(item T) {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	wq.queue.Enqueue(item)
	wq.signal.Signal()
}

// Dequeue removes an item from the queue and blocks until
// an item is available. If the queue is empty, it waits
// until an item is added to the queue.
func (wq *WaitQueue[T]) Dequeue() T {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	for wq.queue.Len() == 0 {
		wq.signal.Wait()
	}
	return wq.queue.Dequeue()
}

// Peek returns the item at the front of the queue without removing it.
// If the queue is empty, it returns the zero value of T.
func (wq *WaitQueue[T]) Peek() T {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	if wq.queue.Len() == 0 {
		return Zero[T]()
	}
	return wq.queue.Peek()
}

// Len returns the number of items in the queue.
func (wq *WaitQueue[T]) Len() int {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	return wq.queue.Len()
}

// IsEmpty returns true if the queue is empty.
func (wq *WaitQueue[T]) IsEmpty() bool {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	return wq.queue.IsEmpty()
}

// Remove calls remove on the underlying queue if it exists.
// If the queue does not support remove, it returns false.
// It is safe to call this method concurrently.
func (wq *WaitQueue[T]) Remove(i int) bool {
	if hasRemove, ok := wq.queue.(interface{ Remove(int) bool }); ok {
		wq.lock.Lock()
		defer wq.lock.Unlock()
		return hasRemove.Remove(i)
	}
	return false
}

// Update tells the queue that a value has been changed in a way that would
// affect its order in the queue. True is returned if the value was updated.
// If the queue does not support update, it returns false.
// It is safe to call this method concurrently.
func (wq *WaitQueue[T]) Update(i int) bool {
	if hasUpdate, ok := wq.queue.(interface{ Update(int) bool }); ok {
		wq.lock.Lock()
		defer wq.lock.Unlock()
		return hasUpdate.Update(i)
	}
	return false
}

// Refreshes the order of the items in the priority queue if they've
// become out of order. This is a no-op if the queue is empty.
// If the queue does not support refresh, it does nothing.
// It is safe to call this method concurrently.
func (wq *WaitQueue[T]) Refresh() {
	if hasRefresh, ok := wq.queue.(interface{ Refresh() }); ok {
		wq.lock.Lock()
		defer wq.lock.Unlock()
		hasRefresh.Refresh()
	}
}

// Values returns a sequence of the values in the queue.
// The order of the values matches the order of the internal queue.
func (wq *WaitQueue[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		wq.lock.Lock()
		defer wq.lock.Unlock()
		for v := range wq.queue.Values() {
			if !yield(v) {
				break
			}
		}
	}
}

// A ready queue is a thread-safe queue where the Dequeue method
// blocks until an item is available and meets a ready state.
// The ready state is a function that returns a value that can be
// used to compare the item at the front of the queue to the ready state.
// As soon as the item at the front of the queue is less than
// the ready state, it is removed from the queue and returned.
type ReadyQueue[T any] struct {
	priority       *PriorityQueue[T]
	wait           *WaitQueue[T]
	readyState     func() T
	checkFrequency time.Duration
}

var _ Queue[any] = &ReadyQueue[any]{}

// NewReadyQueue creates a new ReadyQueue with the given less function,
// ready state function, and check frequency.
func NewReadyQueue[T any](less Less[T], readyState func() T, checkFrequency time.Duration) *ReadyQueue[T] {
	pq := NewPriorityQueue(less)
	wq := NewWaitQueue(pq)

	return &ReadyQueue[T]{
		priority:       pq,
		wait:           wq,
		readyState:     readyState,
		checkFrequency: checkFrequency,
	}
}

// SetIndex sets the function that will be used to set the index of
// the items in the queue.
func (rq *ReadyQueue[T]) SetIndex(setIndex func(T, int)) {
	rq.wait.lock.Lock()
	defer rq.wait.lock.Unlock()
	rq.priority.SetIndex(setIndex)
}

// SetLess sets the less function that will be used to order the items
// in the queue. This will cause the queue to be re-ordered.
func (rq *ReadyQueue[T]) SetLess(less Less[T]) {
	rq.wait.lock.Lock()
	defer rq.wait.lock.Unlock()
	rq.priority.SetLess(less)
}

// GetLess returns the less function that is used to order the items
// in the queue.
func (rq *ReadyQueue[T]) GetLess() Less[T] {
	return rq.priority.GetLess()
}

// Enqueue adds an item to the queue and signals any waiting
// goroutines that an item is available.
func (rq *ReadyQueue[T]) Enqueue(item T) {
	rq.wait.Enqueue(item)
}

// Dequeue removes an item from the queue and blocks until
// an item is available. If the queue is empty, it waits
// until an item is added to the queue. If the item at the front
// of the queue is not less than the ready state, it waits until it
// is checking at the given check frequency.
func (rq *ReadyQueue[T]) Dequeue() T {
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

// Peek returns the item at the front of the queue without removing it.
// If the queue is empty, it returns the zero value of T.
func (rq *ReadyQueue[T]) Peek() T {
	return rq.wait.Peek()
}

// Len returns the number of items in the queue.
func (rq *ReadyQueue[T]) Len() int {
	return rq.wait.Len()
}

// IsEmpty returns true if the queue is empty.
func (rq *ReadyQueue[T]) IsEmpty() bool {
	return rq.wait.IsEmpty()
}

// Values returns a sequence of the values in the queue.
// The order of the values matches the order of the internal queue.
func (rq *ReadyQueue[T]) Values() iter.Seq[T] {
	return rq.wait.Values()
}

// Remove tries to remove the value at the index from the queue.
// True is returned if the value is removed.
func (rq *ReadyQueue[T]) Remove(i int) bool {
	rq.wait.lock.Lock()
	defer rq.wait.lock.Unlock()
	return rq.priority.Remove(i)
}

// Update tells the queue that a value has been changed in a way that would
// affect its order in the queue. True is returned if the value was updated.
func (rq *ReadyQueue[T]) Update(i int) bool {
	rq.wait.lock.Lock()
	defer rq.wait.lock.Unlock()
	return rq.priority.Update(i)
}

// Refreshes the order of the items in the priority queue if they've
// become out of order. This is a no-op if the queue is empty.
func (rq *ReadyQueue[T]) Refresh() {
	rq.wait.lock.Lock()
	defer rq.wait.lock.Unlock()
	rq.priority.Refresh()
}

// A thread-safe queue implementation.
type SyncQueue[T any] struct {
	queue Queue[T]
	lock  sync.Mutex
}

var _ Queue[any] = &SyncQueue[any]{}

// NewSyncQueue creates a new SyncQueue with the given queue.
// The queue is wrapped in a mutex to ensure that all operations
// are thread-safe.
func NewSyncQueue[T any](queue Queue[T]) *SyncQueue[T] {
	return &SyncQueue[T]{
		queue: queue,
		lock:  sync.Mutex{},
	}
}

// Enqueue adds an item to the queue.
// It is safe to call this method concurrently.
func (sq *SyncQueue[T]) Enqueue(item T) {
	sq.lock.Lock()
	defer sq.lock.Unlock()
	sq.queue.Enqueue(item)
}

// Dequeue removes an item from the queue.
// If the queue is empty, a zero value of T is returned.
// It is safe to call this method concurrently.
func (sq *SyncQueue[T]) Dequeue() T {
	sq.lock.Lock()
	defer sq.lock.Unlock()
	return sq.queue.Dequeue()
}

// Pop removes the item at the end of the queue.
// If the queue is empty, it returns the zero value of T.
// It is safe to call this method concurrently.
func (sq *SyncQueue[T]) Peek() T {
	sq.lock.Lock()
	defer sq.lock.Unlock()
	return sq.queue.Peek()
}

// Len returns the number of items in the queue.
// It is safe to call this method concurrently.
func (sq *SyncQueue[T]) Len() int {
	sq.lock.Lock()
	defer sq.lock.Unlock()
	return sq.queue.Len()
}

// IsEmpty returns true if the queue is empty.
// It is safe to call this method concurrently.
func (sq *SyncQueue[T]) IsEmpty() bool {
	sq.lock.Lock()
	defer sq.lock.Unlock()
	return sq.queue.IsEmpty()
}

// Remove calls remove on the underlying queue if it exists.
// If the queue does not support remove, it returns false.
// It is safe to call this method concurrently.
func (sq *SyncQueue[T]) Remove(i int) bool {
	if hasRemove, ok := sq.queue.(interface{ Remove(int) bool }); ok {
		sq.lock.Lock()
		defer sq.lock.Unlock()
		return hasRemove.Remove(i)
	}
	return false
}

// Update tells the queue that a value has been changed in a way that would
// affect its order in the queue. True is returned if the value was updated.
// If the queue does not support update, it returns false.
// It is safe to call this method concurrently.
func (sq *SyncQueue[T]) Update(i int) bool {
	if hasUpdate, ok := sq.queue.(interface{ Update(int) bool }); ok {
		sq.lock.Lock()
		defer sq.lock.Unlock()
		return hasUpdate.Update(i)
	}
	return false
}

// Refreshes the order of the items in the priority queue if they've
// become out of order. This is a no-op if the queue is empty.
// If the queue does not support refresh, it does nothing.
// It is safe to call this method concurrently.
func (sq *SyncQueue[T]) Refresh() {
	if hasRefresh, ok := sq.queue.(interface{ Refresh() }); ok {
		sq.lock.Lock()
		defer sq.lock.Unlock()
		hasRefresh.Refresh()
	}
}

// Values returns a sequence of the values in the queue.
// The order of the values matches the order of the internal queue.
// It locks the queue while iterating over the values.
// It is not safe to modify the queue while iterating.
func (sq *SyncQueue[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		sq.lock.Lock()
		defer sq.lock.Unlock()
		for v := range sq.queue.Values() {
			if !yield(v) {
				return
			}
		}
	}
}
