package httpserver

import (
	"sync"
	"time"
)

const (
	pairCreateBurst = 5
	pairCreateRate  = 5.0 / 60.0 // tokens per second
	shareBurst      = 30
	shareRate       = 30.0 / 60.0
)

type tokenBucket struct {
	mu     sync.Mutex
	burst  float64
	rate   float64
	tokens map[string]bucketState
}

type bucketState struct {
	tokens float64
	last   time.Time
}

func newTokenBucket(burst int, rate float64) *tokenBucket {
	return &tokenBucket{
		burst:  float64(burst),
		rate:   rate,
		tokens: make(map[string]bucketState),
	}
}

func (b *tokenBucket) allow(key string, now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	st := b.tokens[key]
	if st.last.IsZero() {
		st.tokens = b.burst
		st.last = now
	} else {
		elapsed := now.Sub(st.last).Seconds()
		if elapsed > 0 {
			st.tokens += elapsed * b.rate
			if st.tokens > b.burst {
				st.tokens = b.burst
			}
			st.last = now
		}
	}
	if st.tokens < 1 {
		b.tokens[key] = st
		return false
	}
	st.tokens -= 1
	b.tokens[key] = st
	return true
}
