package dsa

import (
	"iter"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// EndOfTime is a constant representing the end of time.
	EndOfTime = time.Unix(1<<63-62135596801, 999999999)
	// BeginningOfTime is a constant representing the beginning of time.
	BeginningOfTime = time.Unix(-62135596801, 0)
)

// A schedule is a collection of events occurring at specific times.
// You can Eval() to get events that are due, or you can call Run() to
// loop sending events to a callback function until a done channel is closed.
// The schedule is thread-safe and can be used from multiple goroutines.
type Schedule[T any] struct {
	getTime      func(T) time.Time
	events       *LinkedList[T]
	eventLock    sync.Mutex
	nextTime     time.Time
	nextTimeLock sync.Mutex
	timer        *time.Timer
	interrupt    chan struct{}
	running      atomic.Bool
	stopOnDone   atomic.Bool
}

// NewSchedule creates a new Schedule with the given function to get the time for each event.
// The getTime function should return the time at which the event is scheduled to occur.
// The schedule is initialized with the next time set to EndOfTime.
// If the event does not have a time (yet), it should return EndOfTime.
// If an event's scheduled time changes,
func NewSchedule[T any](getTime func(T) time.Time) *Schedule[T] {
	return &Schedule[T]{
		events:    NewLinkedList[T](),
		getTime:   getTime,
		nextTime:  EndOfTime,
		interrupt: make(chan struct{}, 1),
		timer:     time.NewTimer(0),
	}
}

// Add adds an event to the schedule and returns a function to remove it.
// The remove function should be called when the event is no longer needed.
func (s *Schedule[T]) Add(event T) (remove func()) {
	s.eventLock.Lock()
	defer s.eventLock.Unlock()

	node := s.events.Add(event)
	remove = func() {
		s.eventLock.Lock()
		defer s.eventLock.Unlock()

		node.Remove()
	}

	s.Update(event)

	return remove
}

// NextTime returns the next scheduled time for the events in the schedule.
// It returns EndOfTime if there are no scheduled events.
func (s *Schedule[T]) NextTime() time.Time {
	return s.nextTime
}

// LastTime returns the last scheduled time for the events in the schedule.
// It returns BeginningOfTime if there are no scheduled events.
// This is a blocking call and iterates over all events.
func (s *Schedule[T]) LastTime() time.Time {
	s.eventLock.Lock()
	defer s.eventLock.Unlock()

	lastTime := BeginningOfTime

	if s.events.IsEmpty() {
		return lastTime
	}

	for event := range s.events.Values() {
		t := s.getTime(event)
		if t.After(lastTime) {
			lastTime = t
		}
	}

	return lastTime
}

// SetNextTime sets the next scheduled time for the events in the schedule.
// It also resets the timer to trigger at the new time.
// If the given time is after the earlest event time, it will schedule
// the events late. If the given time is in the past, it will schedule
// an Eval() in the Run() loop immediately.
func (s *Schedule[T]) SetNextTime(nextTime time.Time) {
	s.nextTimeLock.Lock()
	defer s.nextTimeLock.Unlock()

	s.nextTime = nextTime
	s.Interrupt()
	s.timer.Reset(time.Until(nextTime))
}

// Update updates the schedule based on the new time for the event.
// If the time of the event is more recent than the next time the schedule
// is set to trigger, it will update the next time.
func (s *Schedule[T]) Update(event T) {
	eventTime := s.getTime(event)
	if eventTime.Before(s.nextTime) {
		s.SetNextTime(eventTime)
	}
}

// Eval evaluates the schedule and returns a list of events that are ready
// (i.e., their scheduled time has passed) and the next earliest time for any remaining events.
// This is a blocking call and iterates over all events.
func (s *Schedule[T]) Eval() (ready []T, nextEarliest time.Time) {
	s.eventLock.Lock()
	defer s.eventLock.Unlock()

	now := time.Now()
	nextEarliest = EndOfTime
	for node := range s.events.Nodes() {
		t := s.getTime(node.Value)
		if !t.After(now) {
			node.Remove()
			ready = append(ready, node.Value)
		} else if t.Before(nextEarliest) {
			nextEarliest = t
		}
	}

	return
}

// Run starts the schedule and continuously evaluates it until the done channel is closed.
// It calls the onReady function with the list of ready events whenever they are available.
// onReady is only called when events are ready and it blocks the schedule so it should
// return as quickly as possible.
//
//	go s.Run(ctx.Done(), func(ready []T) {
//	    // Process ready events
//	})
func (s *Schedule[T]) Run(done <-chan struct{}, onReady func([]T)) {
	s.stopOnDone.Store(false)
	s.running.Store(true)

	for s.running.Load() {
		ready, earliest := s.Eval()
		s.nextTime = earliest

		if len(ready) > 0 {
			onReady(ready)
		}

		if earliest.Equal(EndOfTime) && s.stopOnDone.Load() {
			s.running.Store(false)
			return
		}

		s.SetNextTime(earliest)
		s.CancelInterrupt()

		select {
		case <-done:
			return
		case <-s.interrupt:
			// Interrupt signal received, new next time!
		case <-s.timer.C:
			// Timer expired, continue to the next iteration
		}
	}
}

// Stop stops the schedule. If immediate is true, it stops immediately.
// If false, it will stop after all events are processed.
func (s *Schedule[T]) Stop(immediate bool) {
	if immediate {
		s.running.Store(false)
		s.timer.Reset(0)
	} else {
		s.stopOnDone.Store(true)
	}
	s.Interrupt()
}

// CancelInterrupt cancels the interrupt signal if it has been sent.
// This is a non-blocking call and will not wait for the schedule to process the signal.
// It is used to prevent the schedule from being interrupted if it is not needed.
func (s *Schedule[T]) CancelInterrupt() {
	select {
	case <-s.interrupt:
	default:
	}
}

// Interrupt sends an interrupt signal to the schedule.
// This is a non-blocking call and will not wait for the schedule to process the signal.
// It is used to wake up the schedule if it is waiting on a timer.
func (s *Schedule[T]) Interrupt() {
	select {
	case s.interrupt <- struct{}{}:
	default:
	}
}

// IsRunning checks if the schedule is currently running.
func (s *Schedule[T]) IsRunning() bool {
	return s.running.Load()
}

// GetTime returns the scheduled time for the given event based on the time
// function provided during the creation of the schedule.
func (s *Schedule[T]) GetTime(event T) time.Time {
	return s.getTime(event)
}

// IsEmpty checks if the schedule is empty. This is a blocking call.
func (s *Schedule[T]) IsEmpty() bool {
	s.eventLock.Lock()
	defer s.eventLock.Unlock()

	return s.events.IsEmpty()
}

// Len returns the number of events in the schedule. This is a blocking call.
// This is O(n) because it has to iterate the list.
func (s *Schedule[T]) Len() int {
	s.eventLock.Lock()
	defer s.eventLock.Unlock()

	return s.events.Len()
}

// Values returns a sequence of all events in the schedule.
// This is a blocking call and should be used with caution in a multi-threaded environment.
func (s *Schedule[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		s.eventLock.Lock()
		defer s.eventLock.Unlock()

		for event := range s.events.Values() {
			if !yield(event) {
				break
			}
		}
	}
}

// Nodes returns a sequence of all nodes in the schedule.
// Iterating over the nodes allows you to remove any events from the schedule.
// This is a blocking call and should be used with caution in a multi-threaded environment.
func (s *Schedule[T]) Nodes() iter.Seq[*LinkedNode[T]] {
	return func(yield func(*LinkedNode[T]) bool) {
		s.eventLock.Lock()
		defer s.eventLock.Unlock()

		for node := range s.events.Nodes() {
			if !yield(node) {
				break
			}
		}
	}
}
