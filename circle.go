package dsa

import "iter"

// A size bounded stack and queue implementation that uses a circular buffer.
type Circle[T any] struct {
	circle      []T
	head        int
	size        int
	onOverwrite func(T)
}

var _ Queue[any] = &Circle[any]{}
var _ Stack[any] = &Circle[any]{}

// NewCircle creates a new Circle with the given size.
// The size is the maximum number of items that can be stored in the queue.
// If more items are added, the oldest items are removed.
func NewCircle[T any](max int) *Circle[T] {
	return &Circle[T]{
		circle:      make([]T, max),
		onOverwrite: func(value T) {},
	}
}

// SetOnOverwrite sets the function to be called when an item is overwritten.
func (cq *Circle[T]) SetOnOverwrite(fn func(T)) {
	if fn == nil {
		cq.onOverwrite = func(value T) {}
	} else {
		cq.onOverwrite = fn
	}
}

// Enqueue adds an item to the circle.
// If the circle is full, the oldest item is removed.
func (cq *Circle[T]) Enqueue(value T) {
	if cq.size == cq.Cap() {
		cq.onOverwrite(cq.circle[cq.head])
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
