# dsa
A Go module with my common data structures and algorithms.

> go get github.com/clickermonkey/dsa

### Interfaces
- `dsa.Stack[T]` a stack interface (FILO/LIFO)
- `dsa.Queue[T]` a queue interface (FIFO/LILO)

### Concrete
- `dsa.LinkedList[T]` a doubly linked list of T values. non-sequential memory layout but O(1) insertion & removal. Implements stack & queue.
- `dsa.LinkedNode[T]` a doubly linked list node.
- `dsa.Circle[T]` a size bounded stack & queue.
- `dsa.PriorityQueue[T]` a queue where highest priority value is in the front.
- `dsa.WaitQueue[T]` a concurrent queue implementation where `Dequeue()` waits for a value.
- `dsa.ReadyQueue[T]` a concurrent priority queue implementation where dequeue waits for a value which meets an expected priority (ready state).
- `dsa.SyncQueue[T]` a conccurrent queue wrapper.
- `dsa.SliceStack[T]` a slice stack implementation.
- `dsa.WaitStack[T]` a concurrent stack implementation where `Pop()` waits for a value.
- `dsa.SyncStack[T]` a concurrent stack wrapper.

### Helpers
- `dsa.Stopper` a wait/stop object to simplify gates across goroutines.

### Functions
- `dsa.After(fn)` converts a function call into a channel that resolves when the function returns.
- `dsa.ToChannel(fn)` converts a function which returns a value into a channel which resolves that value when the function returns it.
- `dsa.WorkerPool(ctx,n,q,do)` creates an async pool of workers which do work off of a queue which can be stopped at any time.
- `dsa.Executor(ctx,q,do)` creates an async work processor from a queue of work that can be stopped at any time.
