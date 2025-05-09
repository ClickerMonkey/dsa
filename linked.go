package dsa

import "iter"

// A linked list node that can be inserted and removed from anywhere in a list in O(1) time.
type LinkedNode[T any] struct {
	Value T

	next *LinkedNode[T]
	prev *LinkedNode[T]
}

// NewLinkedNode Creates a new linked node with the given value.
func NewLinkedNode[T any](value T) *LinkedNode[T] {
	ln := &LinkedNode[T]{
		Value: value,
	}
	ln.next = ln
	ln.prev = ln
	return ln
}

// Get returns the value of the linked node, or the zero value if the node is nil.
func (ln *LinkedNode[T]) Get() T {
	if ln == nil {
		return Zero[T]()
	}
	return ln.Value
}

// Next returns the next node in the list, or nil if the node is removed.
func (ln *LinkedNode[T]) Next() *LinkedNode[T] {
	if ln.IsRemoved() {
		return nil
	}
	return ln.next
}

// Prev returns the previous node in the list, or nil if the node is removed.
func (ln *LinkedNode[T]) Prev() *LinkedNode[T] {
	if ln.IsRemoved() {
		return nil
	}
	return ln.prev
}

// IsRemoved returns true if the node is removed from the list.
func (ln *LinkedNode[T]) IsRemoved() bool {
	return ln.next == ln
}

// detach removes the node by pointing it's previous and next nodes to each other.
func (ln *LinkedNode[T]) detach() {
	ln.prev.next = ln.next
	ln.next.prev = ln.prev
}

// attach updates the previous and next nodes to point to this node.
func (ln *LinkedNode[T]) attach() {
	ln.prev.next = ln
	ln.next.prev = ln
}

// remove removes the node from the list by detaching it from its previous and next nodes.
func (ln *LinkedNode[T]) Remove() {
	ln.detach()
	ln.next = ln
	ln.prev = ln
}

// InsertAfter inserts the node after the given node in the list.
func (ln *LinkedNode[T]) InsertAfter(after *LinkedNode[T]) {
	ln.detach()
	ln.next = after.next
	ln.prev = after
	ln.attach()
}

// InsertBefore inserts the node before the given node in the list.
func (ln *LinkedNode[T]) InsertBefore(before *LinkedNode[T]) {
	ln.detach()
	ln.prev = before.prev
	ln.next = before
	ln.attach()
}

// A doubly linked list that can be used as a queue or stack.
// This is not a thread safe list, so it should only be used in a single goroutine.
type LinkedList[T any] struct {
	head *LinkedNode[T]
}

var _ Queue[any] = &LinkedList[any]{}
var _ Stack[any] = &LinkedList[any]{}

// Creates a new linked list.
func NewLinkedList[T any]() *LinkedList[T] {
	return &LinkedList[T]{
		head: NewLinkedNode(Zero[T]()),
	}
}

// Adds a new value to the list and returns the new node.
func (ll *LinkedList[T]) Add(value T) *LinkedNode[T] {
	n := NewLinkedNode(value)
	n.InsertBefore(ll.head)
	return n
}

// Dequeue removes the value in the beginning of the list and returns it.
func (ll *LinkedList[T]) Dequeue() T {
	return ll.DequeueNode().Get()
}

// DequeueNode removes the node in the beginning of the list and returns it.
func (ll *LinkedList[T]) DequeueNode() *LinkedNode[T] {
	if ll.IsEmpty() {
		return nil
	}
	first := ll.head.next
	first.Remove()
	return first
}

// Enqueue adds a new value to the end of the list.
func (ll *LinkedList[T]) Enqueue(value T) {
	ll.EnqueueNode(NewLinkedNode(value))
}

// EnqueueNode adds a new node to the end of the list.
func (ll *LinkedList[T]) EnqueueNode(n *LinkedNode[T]) {
	if ll.head != n {
		n.InsertBefore(ll.head)
	}
}

// Push adds a new value to the end of the list.
func (ll *LinkedList[T]) Push(value T) {
	ll.PushNode(NewLinkedNode(value))
}

// PushNode adds a new node to the end of the list.
func (ll *LinkedList[T]) PushNode(n *LinkedNode[T]) {
	if ll.head != n {
		n.InsertBefore(ll.head)
	}
}

// Pop removes the value from the end of the list and returns it.
func (ll *LinkedList[T]) Pop() T {
	return ll.PopNode().Get()
}

// PopNode removes the node from the end of the list and returns it.
func (ll *LinkedList[T]) PopNode() *LinkedNode[T] {
	if ll.IsEmpty() {
		return nil
	}
	popped := ll.head.prev
	popped.Remove()
	return popped
}

// ToFront moves the node to the front of this list.
func (ll *LinkedList[T]) ToFront(n *LinkedNode[T]) {
	n.InsertAfter(ll.head)
}

// ToBack moves the node to the back of this list.
func (ll *LinkedList[T]) ToBack(n *LinkedNode[T]) {
	n.InsertAfter(ll.head.prev)
}

// Peek returns the first value in the list, or the zero value if the list is empty.
func (ll *LinkedList[T]) Peek() T {
	if ll.IsEmpty() {
		return Zero[T]()
	}
	return ll.head.next.Value
}

// Top returns the last value in the list, or the zero value if the list is empty.
func (ll *LinkedList[T]) Top() T {
	if ll.IsEmpty() {
		return Zero[T]()
	}
	return ll.head.prev.Value
}

// Clear removes all values from the list.
func (ll *LinkedList[T]) Clear() {
	ll.head.Remove()
}

// IsEmpty returns true if the list is empty.
func (ll *LinkedList[T]) IsEmpty() bool {
	return ll.head.IsRemoved()
}

// Len returns the number of values in the list.
// This is O(n) because it has to iterate the list.
func (ll *LinkedList[T]) Len() int {
	n := ll.head.next
	size := 0
	for n != ll.head {
		size++
		n = n.next
	}
	return size
}

// Nodes returns an iterator of the nodes in the list.
func (ll *LinkedList[T]) Nodes() iter.Seq[*LinkedNode[T]] {
	return func(yield func(*LinkedNode[T]) bool) {
		n := ll.head.next
		for n != ll.head {
			if !yield(n) {
				break
			}
			n = n.next
		}
	}
}

// Values returns an iterator of the values in the list.
func (ll *LinkedList[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		n := ll.head.next
		for n != ll.head {
			if !yield(n.Value) {
				break
			}
			n = n.next
		}
	}
}
