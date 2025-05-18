package dsa_test

import (
	"testing"
	"time"

	"github.com/clickermonkey/dsa"
)

func TestLinked(t *testing.T) {
	ll := dsa.NewLinkedList[int]()
	ll.Add(1)
	n2 := ll.Add(2)
	n3 := ll.Add(3)

	assertMatches := func(ints ...int) {
		if len(ints) != ll.Len() {
			t.Fatalf("expected length %d, got %d", len(ints), ll.Len())
		}
		i := 0
		for value := range ll.Values() {
			if value != ints[i] {
				t.Fatalf("expected %d at %d, got %d", ints[i], i, value)
			}
			i++
		}
	}

	assertMatches(1, 2, 3)
	n2.Remove()
	assertMatches(1, 3)
	ll.ToFront(n3)
	assertMatches(3, 1)
	if popped := ll.Pop(); popped != 1 {
		t.Fatalf("popped expected 1, got %d", popped)
	}
	assertMatches(3)
	if popped := ll.Pop(); popped != 3 {
		t.Fatalf("popped expected 3, got %d", popped)
	}
	assertMatches()
	if popped := ll.Pop(); popped != 0 {
		t.Fatalf("popped expected 0, got %d", popped)
	}
}

type job struct {
	name string
	at   time.Time
}

func nowJob() *job {
	return &job{
		at: time.Now(),
	}
}
func firstJob(a, b *job) bool {
	return a.at.Before(b.at)
}

func TestReadyQueue(t *testing.T) {
	checkFrequency := time.Millisecond * 2
	rq := dsa.NewReadyQueue(firstJob, nowJob, checkFrequency)

	start := time.Now()

	rq.Enqueue(&job{name: "1 (5)", at: start.Add(time.Millisecond * 5)})
	rq.Enqueue(&job{name: "2 (15)", at: start.Add(time.Millisecond * 15)})
	rq.Enqueue(&job{name: "3 (10)", at: start.Add(time.Millisecond * 10)})

	go func() {
		time.Sleep(time.Millisecond * 6)
		rq.Enqueue(&job{name: "4 (6)", at: time.Now()})
	}()

	order := dsa.NewLinkedList[string]()
	order.Enqueue("1 (5)")
	order.Enqueue("4 (6)")
	order.Enqueue("3 (10)")
	order.Enqueue("2 (15)")

	for !rq.IsEmpty() {
		next := rq.Dequeue()
		now := time.Now()
		fromAt := now.Sub(next.at)

		expected := order.Dequeue()
		if expected != next.name {
			t.Fatalf("job %s expected, got %s", expected, next.name)
		}

		if fromAt >= checkFrequency*2 {
			t.Fatalf("job %s took too long to dequeue %s", next.name, fromAt)
		}
	}
}

type indexedJob struct {
	job
	index int
}

func (ij *indexedJob) Index() *int {
	return &ij.index
}

func firstIndexedJob(a, b *indexedJob) bool {
	return a.at.Before(b.at)
}

func TestDynamicQueue(t *testing.T) {
	rq := dsa.NewPriorityQueue(firstIndexedJob)
	rq.SetIndex(func(ij *indexedJob, i int) {
		ij.index = i
	})

	assertMatches := func(names ...string) {
		if len(names) != rq.Len() {
			t.Fatalf("expected length %d, got %d", len(names), rq.Len())
		}
		i := 0
		for value := range rq.Values() {
			if value.name != names[i] {
				t.Fatalf("expected %s at %d, got %s", names[i], i, value.name)
			}
			i++
		}
	}

	start := time.Now()

	j1 := &indexedJob{job{name: "1 (5)", at: start.Add(time.Millisecond * 5)}, -1}
	j2 := &indexedJob{job{name: "2 (15)", at: start.Add(time.Millisecond * 15)}, -1}
	j3 := &indexedJob{job{name: "3 (10)", at: start.Add(time.Millisecond * 10)}, -1}

	assertMatches()

	rq.Enqueue(j1)
	assertMatches("1 (5)")

	rq.Enqueue(j2)
	assertMatches("1 (5)", "2 (15)")

	rq.Enqueue(j3)
	assertMatches("1 (5)", "3 (10)", "2 (15)")

	j2.at = start.Add(time.Millisecond * 7)
	rq.Update(j2.index)
	assertMatches("1 (5)", "2 (15)", "3 (10)")

	rq.Remove(j2.index)
	assertMatches("1 (5)", "3 (10)")

	rq.Remove(j3.index)
	assertMatches("1 (5)")

	rq.Remove(j1.index)
	assertMatches()
}
