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
)

type randomElectionTimer struct {
	debug  bool
	timer  *time.Timer
	resetC chan bool
}

// NewElectionTimer returns a running electionTimer instance
func NewElectionTimer(f func(), debug bool) ResettingTimer {
	t := randomElectionTimer{debug: debug, resetC: make(chan bool)}
	t.timer = time.AfterFunc(t.electionTimeout(), func() {
		go f()
		t.reset()
	})
	go t.watchChan()
	return &t
}

func (t *randomElectionTimer) Stop() {
	t.timer.Stop()
}
func (t *randomElectionTimer) reset() {
	t.resetC <- true
}

func (t *randomElectionTimer) watchChan() {
	for {
		<-t.resetC
		t.timer.Reset(t.electionTimeout())
	}
}

func (t *randomElectionTimer) Reset() {
	t.reset()
}

func (t *randomElectionTimer) electionTimeout() time.Duration {
	if t.debug {
		return RandomElectionTimeout() * 100
	}
	return RandomElectionTimeout()
}

// RandomElectionTimeout returns a randomized election timeout between 150 and 300ms
func RandomElectionTimeout() time.Duration {
	jitter := rand.Float32() * (maxElectionTimeout - minElectionTimeout)
	return time.Duration(minElectionTimeout+jitter) * time.Millisecond
}

type heartbeatTimer struct {
	timeout time.Duration
	timer   *time.Timer
}

const HeartbeatTimeout = 15 * time.Millisecond // 0.5 to 20ms

func NewHeartbeatTimer(f func()) ResettingTimer {
	t := heartbeatTimer{timeout: HeartbeatTimeout}
	t.timer = time.AfterFunc(HeartbeatTimeout, func() {
		t.Reset()
		f()
	})
	return &t
}

func (t *heartbeatTimer) Stop() {
	t.timer.Stop()
}

func (t *heartbeatTimer) Reset() {
	// This is only correct for timers created with time.AfterFunc
	t.timer.Stop()
	t.timer.Reset(t.timeout)
}

type dummyElectionTimer int

// NewDummyElectionTimer returns a noop ResettingTimer instance for testing
func NewDummyElectionTimer() ResettingTimer {
	t := dummyElectionTimer(0)
	return &t
}

func (t *dummyElectionTimer) Stop() {

}

func (t *dummyElectionTimer) Reset() {

}
