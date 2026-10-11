package agent

import "testing"

func TestShouldRetry(t *testing.T) {
	cases := []struct {
		status int
		want   bool
	}{
		{0, true},
		{429, true},
		{500, true},
		{503, true},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{409, false},
		{410, false},
		{204, false},
	}
	for _, tc := range cases {
		if got := ShouldRetry(tc.status); got != tc.want {
			t.Fatalf("status %d: got %v want %v", tc.status, got, tc.want)
		}
	}
}

func TestFullJitter_bounded(t *testing.T) {
	for attempt := 1; attempt <= 8; attempt++ {
		d := FullJitter(attempt, 200, 1000)
		if d < 0 || d > 1000 {
			t.Fatalf("attempt %d: %d", attempt, d)
		}
	}
}
