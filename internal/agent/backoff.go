package agent

import (
	"math/rand"
	"time"
)

func FullJitter(attempt int, base, cap time.Duration) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := base
	for i := 1; i < attempt; i++ {
		if d > cap/2 {
			d = cap
			break
		}
		d *= 2
	}
	if d > cap {
		d = cap
	}
	if d <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(d) + 1))
}

func ShouldRetry(status int) bool {
	return status == 0 || status == 429 || status >= 500
}
