package dsa

import "iter"

type LinkedNode[T any] struct {
	Value T

	next *LinkedNode[T]
	prev *LinkedNode[T]
}

func NewLinkedNode[T any](value T) *LinkedNode[T] {
	ln := &LinkedNode[T]{
		Value: value,
	}
	ln.next = ln
	ln.prev = ln
	return ln
}

func (ln *LinkedNode[T]) Get() T {
	if ln == nil {
		return zero[T]()
	}
	return ln.Value
}

func (ln *LinkedNode[T]) Next() *LinkedNode[T] {
	if ln.IsRemoved() {
		return nil
	}
	return ln.next
}

func (ln *LinkedNode[T]) Prev() *LinkedNode[T] {
	if ln.IsRemoved() {
		return nil
	}
	return ln.prev
}

func (ln *LinkedNode[T]) IsRemoved() bool {
	return ln.next == ln
}

func (ln *LinkedNode[T]) detach() {
	ln.prev.next = ln.next
	ln.next.prev = ln.prev
}

func (ln *LinkedNode[T]) attach() {
	ln.prev.next = ln
	ln.next.prev = ln
}

func (ln *LinkedNode[T]) Remove() {
	ln.detach()
	ln.next = ln
	ln.prev = ln
}

func (ln *LinkedNode[T]) InsertAfter(after *LinkedNode[T]) {
	ln.detach()
	ln.next = after.next
	ln.prev = after
	ln.attach()
}

func (ln *LinkedNode[T]) InsertBefore(before *LinkedNode[T]) {
	ln.detach()
	ln.prev = before.prev
	ln.next = before
	ln.attach()
}

type LinkedList[T any] struct {
	head *LinkedNode[T]
}

var _ Queue[any] = &LinkedList[any]{}
var _ Stack[any] = &LinkedList[any]{}

func NewLinkedList[T any]() *LinkedList[T] {
	return &LinkedList[T]{
		head: NewLinkedNode(zero[T]()),
	}
}

func (ll *LinkedList[T]) Add(value T) *LinkedNode[T] {
	n := NewLinkedNode(value)
	n.InsertBefore(ll.head)
	return n
}

func (ll *LinkedList[T]) Dequeue() T {
	return ll.DequeueNode().Get()
}

func (ll *LinkedList[T]) DequeueNode() *LinkedNode[T] {
	if ll.IsEmpty() {
		return nil
	}
	first := ll.head.next
	first.Remove()
	return first
}

func (ll *LinkedList[T]) Enqueue(value T) {
	ll.EnqueueNode(NewLinkedNode(value))
}

func (ll *LinkedList[T]) EnqueueNode(n *LinkedNode[T]) {
	if ll.head != n {
		n.InsertBefore(ll.head)
	}
}

func (ll *LinkedList[T]) Push(value T) {
	ll.PushNode(NewLinkedNode(value))
}

func (ll *LinkedList[T]) PushNode(n *LinkedNode[T]) {
	if ll.head != n {
		n.InsertBefore(ll.head)
	}
}

func (ll *LinkedList[T]) Pop() T {
	return ll.PopNode().Get()
}

func (ll *LinkedList[T]) PopNode() *LinkedNode[T] {
	if ll.IsEmpty() {
		return nil
	}
	popped := ll.head.prev
	popped.Remove()
	return popped
}

func (ll *LinkedList[T]) ToFront(n *LinkedNode[T]) {
	n.InsertAfter(ll.head)
}

func (ll *LinkedList[T]) ToBack(n *LinkedNode[T]) {
	n.InsertAfter(ll.head.prev)
}

func (ll *LinkedList[T]) Peek() T {
	if ll.IsEmpty() {
		return zero[T]()
	}
	return ll.head.next.Value
}

func (ll *LinkedList[T]) Top() T {
	if ll.IsEmpty() {
		return zero[T]()
	}
	return ll.head.prev.Value
}

func (ll *LinkedList[T]) Clear() {
	ll.head.Remove()
}

func (ll *LinkedList[T]) IsEmpty() bool {
	return ll.head.IsRemoved()
}

func (ll *LinkedList[T]) Len() int {
	n := ll.head.next
	size := 0
	for n != ll.head {
		size++
		n = n.next
	}
	return size
}

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
