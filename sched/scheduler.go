package sched

import (
	"log"
	"time"
)

type Signal int

const (
	Pause Signal = iota
	Reset
	Resume
	Stop
)

// Schedule starts a timer that runs the provided Schedulable.
// The timer resets itself automatically after the timeout.
// func Schedule(ScheduleFunc, Schedulable, chan Signal) Scheduler

// Scheduler can schedule a Schedulable type and run its RunOnSchedule function
// in an interval defined by ScheduleFunc
type Scheduler struct {
	S chan Signal
}

// ScheduleFunc is used to determine the next time the Schedulable will run
type ScheduleFunc func() time.Duration

// Schedulable defines a type that can be run by the Scheduler.
// RunOnSchedule will be executed every time the Schedule is due.
type Schedulable interface {
	RunOnSchedule()
}

// Schedule ...
func Schedule(f ScheduleFunc, sched Schedulable, signal chan Signal) Scheduler {
	s := Scheduler{signal}

	go schedule(f, sched, signal)

	return s
}

func schedule(f ScheduleFunc, sched Schedulable, signal chan Signal) {
	timeToWait := f()
	select {
	case <-time.After(timeToWait):
		go sched.RunOnSchedule()
		schedule(f, sched, signal)
	case sig := <-signal:
		switch sig {
		case Pause:
			resumeOrStop := waitForResumeOrStop(signal)
			if resumeOrStop == Stop {
				return
			}
			// Resume
			schedule(f, sched, signal)
		case Reset:
			schedule(f, sched, signal)
		case Stop:
			return
		case Resume:
			log.Fatalln("Nothing to resume")
		}
	}
}

func waitForResumeOrStop(signal chan Signal) Signal {
	for {
		select {
		case s := <-signal:
			// Discard all signals except Resume and Stop
			switch s {
			case Resume:
				return Resume
			case Stop:
				return Stop
			default:
				log.Fatalln("Received inappropriate signal on pause", s)
			}
		}
	}
}
