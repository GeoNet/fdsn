package metrics

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

var timers chan Timer

// for aggregating timers
var agg = struct {
	count, sum map[string]int
	taken      map[string][]int
	m          sync.RWMutex
}{
	count: make(map[string]int),
	sum:   make(map[string]int),
	taken: make(map[string][]int),
}

// Timer is for timing events
type Timer struct {
	start   time.Time
	id      string
	taken   int
	stopped bool
}

type TimerStats struct {
	ID           string
	Count        int
	Average      int
	Percentile95 int
	Percentile50 int
}

func init() {
	timers = make(chan Timer, 300)

	go func() {
		for {
			select {
			case m := <-timers:
				agg.m.Lock()
				agg.count[m.id]++
				agg.sum[m.id] += m.taken
				agg.taken[m.id] = append(agg.taken[m.id], m.taken)
				agg.m.Unlock()
			}
		}
	}()
}

// Start returns started Timer.
func Start() Timer {
	return Timer{
		start: time.Now().UTC(),
	}
}

// Stops the timer
func (t *Timer) Stop() {
	t.taken = int(time.Since(t.start) / time.Millisecond)
	t.stopped = true
}

// Stops the timer if it is not already stopped.  Tracks the time taken
// in milliseconds with identity id.
func (t *Timer) Track(id string) error {
	if !t.stopped {
		t.Stop()
	}

	t.id = id

	select {
	case timers <- *t:
	default:
		return fmt.Errorf("failed to track timer %s took %d", t.id, t.taken)
	}

	return nil
}

// Returns the time taken between start and stop in milliseconds.
func (t *Timer) Taken() int {
	return t.taken
}

func ReadTimers() []TimerStats {
	var s []TimerStats
	agg.m.Lock()
	for k, v := range agg.count {
		s = append(s, TimerStats{
			ID:           k,
			Count:        agg.count[k],
			Average:      agg.sum[k] / v,
			Percentile50: percentile(0.5, agg.taken[k]),
			Percentile95: percentile(0.95, agg.taken[k]),
		})

		delete(agg.count, k)
		delete(agg.sum, k)
		delete(agg.taken, k)
	}
	agg.m.Unlock()

	return s
}

// calculates the kth percentile of v
func percentile(k float64, v []int) (value int) {
	if !sort.IntsAreSorted(v) {
		sort.Ints(v)
	}

	p := k * float64(len(v))

	if p != math.Trunc(p) {
		idx := int(math.Ceil(p))
		if idx <= len(v) {
			value = v[int(math.Ceil(p))-1]
		}
	} else {
		idx := int(math.Trunc(p))
		if idx < len(v) {
			value = int((v[idx-1] + v[idx]) / 2)
		}
	}

	return
}
