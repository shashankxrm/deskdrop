package httpserver

import (
	"sync"
	"time"
)

const claimFailWindow = time.Minute
const claimFailLimit = 5 // 6th failed claim in the window is 429

type claimLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func newClaimLimiter() *claimLimiter {
	return &claimLimiter{hits: make(map[string][]time.Time)}
}

func (l *claimLimiter) recordFailure(ip string, now time.Time) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := now.Add(-claimFailWindow)
	kept := l.hits[ip][:0]
	for _, t := range l.hits[ip] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	kept = append(kept, now)
	l.hits[ip] = kept
	return len(kept)
}
