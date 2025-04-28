package dsa

import (
	"fmt"
	"testing"
	"time"
)

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
	rq := NewReadyQueue(firstJob, nowJob, time.Millisecond*2)

	start := time.Now()

	rq.Enqueue(&job{name: "1 (5)", at: start.Add(time.Millisecond * 5)})
	rq.Enqueue(&job{name: "2 (15)", at: start.Add(time.Millisecond * 15)})
	rq.Enqueue(&job{name: "3 (10)", at: start.Add(time.Millisecond * 10)})

	go func() {
		time.Sleep(time.Millisecond * 6)
		rq.Enqueue(&job{name: "4 (6)", at: time.Now()})
	}()

	for rq.Len() > 0 {
		next := rq.Dequeue()
		now := time.Now()
		fromStart := now.Sub(start)
		fromAt := now.Sub(next.at)

		fmt.Printf("job %s dequeued in %s from start, %s from target\n", next.name, fromStart, fromAt)
	}
}
