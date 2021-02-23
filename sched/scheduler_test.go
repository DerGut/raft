package sched

import (
	"testing"
	"time"
)

const waitTimeout = 1 * time.Second

func Test_Scheduler_runsSchedulable(t *testing.T) {
	sched := testSchedulable{hasRun: make(chan struct{})}
	Schedule(zeroDuration, &sched, nil)

	repititions := 3
	for i := 0; i < repititions; i++ {
		select {
		case <-sched.hasRun:
		case <-time.After(waitTimeout):
			t.Errorf("Scheduler hasn't run. Repitition: %d", i)
		}
	}
}

func Test_Scheduler_stopsSchedule(t *testing.T) {
	sched := testSchedulable{hasRun: make(chan struct{})}
	signal := make(chan Signal, 1)
	signal <- Stop
	Schedule(zeroDuration, &sched, signal)

	select {
	case <-sched.hasRun:
		t.Errorf("Scheduler hasn't stopped")
	case <-time.After(waitTimeout): // TODO: something bettern than a timeout?
	}
}

func Test_Scheduler_resetsSchedule(t *testing.T) {
	sched := testSchedulable{hasRun: make(chan struct{})}
	signal := make(chan Signal, 1)
	schedulerHasRun := make(chan struct{}, 1)
	zeroDuration := func() time.Duration {
		schedulerHasRun <- struct{}{}
		return 0
	}
	Schedule(zeroDuration, &sched, signal)

	select {
	case <-schedulerHasRun:
	case <-time.After(waitTimeout):
		t.Error("scheduler hasn't run")
	}
	select {
	case <-sched.hasRun:
	case <-time.After(waitTimeout):
		t.Error("schedulable hasn't run")
	}

	signal <- Reset
	<-schedulerHasRun

	select {
	case <-schedulerHasRun:
	case <-sched.hasRun:
		t.Error("Schedulable shouldn't have run")
	}
}

func Test_Scheduler_pausesAndResumesSchedule(t *testing.T) {
	sched := testSchedulable{hasRun: make(chan struct{})}
	signal := make(chan Signal, 1)
	signal <- Pause
	Schedule(zeroDuration, &sched, signal)

	select {
	case <-time.After(waitTimeout):
	case <-sched.hasRun:
		t.Errorf("Scheduler should have paused")
	}

	signal <- Resume

	select {
	case <-time.After(waitTimeout):
		t.Error("Scheduler should have resumed")
	case <-sched.hasRun:
	}
}

func Test_Scheduler_pausesAndStopsSchedule(t *testing.T) {
	sched := testSchedulable{hasRun: make(chan struct{})}
	signal := make(chan Signal, 1)
	signal <- Pause
	Schedule(zeroDuration, &sched, signal)

	select {
	case <-time.After(waitTimeout):
	case <-sched.hasRun:
		t.Errorf("Scheduler should have paused")
	}

	signal <- Stop

	select {
	case <-time.After(waitTimeout):
	case <-sched.hasRun:
		t.Error("Scheduler shouldn't have run")
	}
}

func zeroDuration() time.Duration { return 0 }

type testSchedulable struct{ hasRun chan struct{} }

func (s *testSchedulable) RunOnSchedule() { s.hasRun <- struct{}{} }
