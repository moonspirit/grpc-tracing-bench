package metrics

import (
	"fmt"
	"sync"
	"time"
)

type LatencyStats struct {
	mu    sync.Mutex
	ts    time.Time
	count int64
	sum   int64
}

func NewLatencyStats() *LatencyStats {
	return &LatencyStats{ts: time.Now()}
}

func (s *LatencyStats) String() string {

	s.mu.Lock()
	count := s.count
	sum := s.sum
	ts := s.ts
	s.count = 0
	s.sum = 0
	s.ts = time.Now()
	s.mu.Unlock()

	if count > 0 {
		return fmt.Sprint("average: ", time.Duration(sum/count), " qps: ", float64(count)/s.ts.Sub(ts).Seconds())
	} else {
		return fmt.Sprint("average: ", 0, " qps: ", 0)
	}
}

func (s *LatencyStats) Observe(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.count++
	s.sum += d.Nanoseconds()
}
