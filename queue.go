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
	items []T
	less  Less[T]
}

var _ Queue[any] = &PriorityQueue[any]{}
var _ heap.Interface = &PriorityQueue[any]{}

// NewPriorityQueue creates a new priority queue with the given less function.
// The less function orders the items in the queue, the first item in the queue
// is the one that is less than all the others.
func NewPriorityQueue[T any](less Less[T]) *PriorityQueue[T] {
	return &PriorityQueue[T]{
		items: make([]T, 0),
		less:  less,
	}
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
	SetIndex(pq.items[i], i)
	SetIndex(pq.items[j], j)
}

// Push adds an item to the queue.
// This is used by the heap interface to add items to the queue.
func (pq *PriorityQueue[T]) Push(x any) {
	item := x.(T)
	SetIndex(item, len(pq.items))
	pq.items = append(pq.items, item)
}

// Pop removes the item at the end of the queue.
// This is used by the heap interface to remove items from the queue.
func (pq *PriorityQueue[T]) Pop() any {
	old := pq.items
	n := len(old)
	item := old[n-1]
	SetIndex(item, -1)
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

// Tries to remove the value from the queue if supported.
// If the value does not have an index false is returned.
// The value must implement dsa.WithIndex, has an index within the bounds
// of the queue, and the item at the index appears to be equal to the given
// value when using the less function on the priority queue.
// True is returned if the value is removed.
func (pq *PriorityQueue[T]) Remove(value T) bool {
	if i := GetIndex(value, -1); i >= 0 && i < len(pq.items) {
		actual := pq.items[i]
		if !LessEqual(actual, value, pq.less) {
			return false
		}

		heap.Remove(pq, i)
		SetIndex(value, -1)
		return true
	}
	return false
}

// Tells the queue that a value has been changed in a way that would
// affect its order in the queue. The value must implement WithIndex.
// If it does not -1 is returned, otherwise the number of moves is returned
// (absolute value of new index - old index).
func (pq *PriorityQueue[T]) Update(value T) int {
	if start := GetIndex(value, -1); start != -1 {
		heap.Fix(pq, start)
		end := GetIndex(value, -1)
		if end > start {
			return end - start
		} else {
			return start - end
		}
	}
	return -1
}

// Refreshes the order of the items in the priority queue if they've
// become out of order.
func (pq *PriorityQueue[T]) Refresh() {
	heap.Init(pq)
}

// Values returns a sequence of the values in the queue.
// The order of the values is not guaranteed.
func (pq *PriorityQueue[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range pq.items {
			if !yield(v) {
				break
			}
		}
	}
}

// OrderedValues returns a sequence of the values in the queue.
// The order of the values is guaranteed to be in the order of least to
// greatest according to the less function.
func (pq *PriorityQueue[T]) OrderedValues() iter.Seq[T] {
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
func (ts *WaitQueue[T]) Len() int {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	return ts.queue.Len()
}

// IsEmpty returns true if the queue is empty.
func (ts *WaitQueue[T]) IsEmpty() bool {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	return ts.queue.IsEmpty()
}

// Values returns a sequence of the values in the queue.
// The order of the values matches the order of the internal queue.
func (ts *WaitQueue[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		ts.lock.Lock()
		defer ts.lock.Unlock()
		for v := range ts.queue.Values() {
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

// OrderedValues returns a sequence of the values in the queue.
// The order of the values is guaranteed to be in the order of least to
// greatest according to the less function.
func (rq *ReadyQueue[T]) OrderedValues() iter.Seq[T] {
	return func(yield func(T) bool) {
		rq.wait.lock.Lock()
		defer rq.wait.lock.Unlock()
		for v := range rq.priority.OrderedValues() {
			if !yield(v) {
				return
			}
		}
	}
}

// Remove tries to remove the value from the queue if supported.
// If the value exists in the queue, implements WithIndex, and has an index
// within the bounds of the queue, it is removed and true is returned.
func (rq *ReadyQueue[T]) Remove(value T) bool {
	rq.wait.lock.Lock()
	defer rq.wait.lock.Unlock()
	return rq.priority.Remove(value)
}

// Update tells the queue that a value has been changed in a way that would
// affect its order in the queue. The value must implement WithIndex.
// If it does not -1 is returned, otherwise the number of moves is returned
// (absolute value of new index - old index).
func (rq *ReadyQueue[T]) Update(value T) int {
	rq.wait.lock.Lock()
	defer rq.wait.lock.Unlock()
	return rq.priority.Update(value)
}

// Refreshes the order of the items in the priority queue if they've
// become out of order. This is a no-op if the queue is empty.
func (rq *ReadyQueue[T]) Refresh() {
	rq.wait.lock.Lock()
	defer rq.wait.lock.Unlock()
	rq.priority.Refresh()
}

// A size bounded stack and queue implementation that uses a circular buffer.
type Circle[T any] struct {
	circle []T
	head   int
	size   int
}

var _ Queue[any] = &Circle[any]{}
var _ Stack[any] = &Circle[any]{}

// NewCircle creates a new Circle with the given size.
// The size is the maximum number of items that can be stored in the queue.
// If more items are added, the oldest items are removed.
func NewCircle[T any](max int) *Circle[T] {
	return &Circle[T]{
		circle: make([]T, max),
	}
}

// Enqueue adds an item to the circle.
// If the circle is full, the oldest item is removed.
func (cq *Circle[T]) Enqueue(value T) {
	if cq.size == cq.Cap() {
		cq.circle[cq.head] = value
		cq.head = cq.next()
	} else {
		cq.circle[cq.tail()] = value
		cq.size++
	}
}

// Push adds an item to the circle.
func (cq *Circle[T]) Push(value T) {
	cq.Enqueue(value)
}

// Dequeue removes the item at the front of the circle.
// If the circle is empty, it returns the zero value of T.
func (cq *Circle[T]) Dequeue() T {
	z := Zero[T]()
	if cq.size == 0 {
		return z
	}
	head := cq.circle[cq.head]
	cq.circle[cq.head] = z
	cq.head = cq.next()
	cq.size--
	return head
}

// Pop removes the item at the end of the circle.
// If the circle is empty, it returns the zero value of T.
func (cq *Circle[T]) Pop() T {
	z := Zero[T]()
	if cq.size == 0 {
		return z
	}
	tailIndex := cq.last()
	tail := cq.circle[tailIndex]
	cq.circle[tailIndex] = z
	cq.size--
	return tail
}

// Peek returns the item at the front of the circle without removing it.
// If the circle is empty, it returns the zero value of T.
func (cq *Circle[T]) Peek() T {
	if cq.size == 0 {
		return Zero[T]()
	}
	return cq.circle[cq.head]
}

// Top returns the item at the end of the circle without removing it.
// If the circle is empty, it returns the zero value of T.
func (cq *Circle[T]) Top() T {
	if cq.size == 0 {
		return Zero[T]()
	}
	return cq.circle[cq.last()]
}

// Len returns the number of items in the circle.
func (cq *Circle[T]) Len() int {
	return cq.size
}

// IsEmpty returns true if the circle is empty.
func (cq *Circle[T]) IsEmpty() bool {
	return cq.size == 0
}

// Values returns a sequence of the values in the circle.
// The order of the values is oldest to newest.
func (cq *Circle[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := range cq.size {
			if !yield(cq.circle[cq.wrap(cq.head+i)]) {
				break
			}
		}
	}
}

// Cap returns the capacity of the circle.
// This is the maximum number of items that can be stored in the circle.
func (cq *Circle[T]) Cap() int {
	return len(cq.circle)
}

// wrap wraps the given index around the circle.
func (cq *Circle[T]) wrap(i int) int {
	n := cq.Cap()
	return (i + n) % n
}

// tail returns the index of the next item to be added to the circle.
func (cq *Circle[T]) tail() int {
	return cq.wrap(cq.head + cq.size)
}

// last returns the index of the last item in the circle.
func (cq *Circle[T]) last() int {
	return cq.wrap(cq.head + cq.size - 1)
}

// next returns the index of the next head of the circle.
func (cq *Circle[T]) next() int {
	return cq.wrap(cq.head + 1)
}
