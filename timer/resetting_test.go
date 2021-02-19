package timer

import (
	"testing"
	"time"
)

func TestRandomElectionTimeoutIsInRange(t *testing.T) {
	for i := 0; i < 100; i++ {
		got := RandomElectionTimeout()
		if got < minElectionTimeout*time.Millisecond || got > maxElectionTimeout*time.Millisecond {
			t.Errorf("RandomElectionTimeout() = %v", got)
		}
	}
}
