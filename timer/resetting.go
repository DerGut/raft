package timer

import (
	"math/rand"
	"time"
)

type ResettingTimer interface {
	Stop()
	Reset()
}

const (
	// or 10 to 500ms
	minElectionTimeout = 150
	maxElectionTimeout = 300

	HeartbeatTimeout = 15 * time.Millisecond // 0.5 to 20ms
)

// RandomElectionTimeout returns a randomized election timeout between 150 and 300ms
func RandomElectionTimeout() time.Duration {
	jitter := rand.Float32() * (maxElectionTimeout - minElectionTimeout)
	return time.Duration(minElectionTimeout+jitter) * time.Millisecond
}
