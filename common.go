package dsa

import (
	"container/heap"
	"iter"
)

// A comparison function that returns true if a is less than b.
type Less[T any] func(a T, b T) bool

// Converts a Less func to an Equal func
func LessToEqual[T any](less Less[T]) Less[T] {
	return func(a, b T) bool {
		return !less(a, b) && !less(b, a)
	}
}

// Converts a Less func to an More func
func LessToMore[T any](less Less[T]) Less[T] {
	return func(a, b T) bool {
		return less(b, a)
	}
}

// Returns whether a and b are equal given a less function.
func LessEqual[T any](a, b T, less Less[T]) bool {
	return !less(a, b) && !less(b, a)
}

// A type that has an index that can be gotten or updated.
// This is useful for containers that store items in a sequential
// way and the item wants to know what index it's at and it's
// also useful for containers which need to know the index for
// doing more efficient operations.
type WithIndex interface {
	Index() *int
}

// Returns the zero value of T.
func Zero[T any]() T {
	var z T
	return z
}

// A type that holds a value and an index.
type Indexed[V any] struct {
	Value V
	index int
}

// Returns a new Indexed type for V.
func NewIndexed[V any](value V) *Indexed[V] {
	return &Indexed[V]{Value: value, index: -1}
}

func (i *Indexed[V]) Index() *int {
	return &i.index
}

// Sets the index on the given value if it's supported.
func SetIndex(v any, i int) bool {
	if hasIndex, ok := v.(WithIndex); ok && hasIndex != nil {
		target := hasIndex.Index()
		if target != nil {
			*target = i
			return true
		}
	}
	return false
}

// Gets the index on the given value if it's supported, otherwise
// returns the given index.
func GetIndex(v any, otherwise int) int {
	if hasIndex, ok := v.(WithIndex); ok && hasIndex != nil {
		target := hasIndex.Index()
		if target != nil {
			return *target
		}
	}
	return otherwise
}

// Iterates a heap interface in order without copying or recursion.
// This is a modified breadth first search that looks at the
// children of the lowest value in the heap and yields them
// in order of their value. This is useful for heaps that
// are not sorted but need to be iterated in order.
func HeapIterate(h heap.Interface) iter.Seq[int] {
	n := h.Len()
	if n == 0 {
		return func(yield func(int) bool) {}
	}

	// front is all nodes in modified breadth first search
	// the breadth in the context of a heap are not items in the same
	// depth but are items all being looked at for the lowest value.
	// the lowest value is found, removed, yielded, and its children are added to the front.
	front := make([]int, (n+1)/2)
	frontSize := 1

	// adds to the front the child nodes of i
	split := func(heapIndex int) {
		if left := heapIndex*2 + 1; left < n {
			front[frontSize] = left
			frontSize++
		}
		if right := heapIndex*2 + 2; right < n {
			front[frontSize] = right
			frontSize++
		}
	}

	// takes the item at the given index in the front
	take := func(frontIndex int) int {
		taken := front[frontIndex]
		frontSize--
		front[frontIndex] = front[frontSize]
		return taken
	}

	// iterate until we've looked at all items on the heap
	return func(yield func(int) bool) {
		for frontSize > 0 {
			chosenIndex := 0
			for i := 1; i < frontSize; i++ {
				if h.Less(front[i], front[chosenIndex]) {
					chosenIndex = i
				}
			}
			chosen := take(chosenIndex)
			if !yield(chosen) {
				return
			}
			split(chosen)
		}
	}
}
