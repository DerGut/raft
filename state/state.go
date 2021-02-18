package state

import (
	"fmt"
	"math"

	"github.com/DerGut/kv-store/replog"
)

type State interface {
	CurrentTerm() replog.Term
	IncrCurrentTerm(id *string)
	UpdateTerm(new replog.Term)
	VotedFor() *string
	CanVoteFor(id string) bool
	SetVotedFor(id string)
	Log() replog.Log
	DeleteConflictingAndAddNewEntries(prevLogIndex int, entries []replog.Entry)
	SetLog(l replog.Log)
	CommitIndex() int
	UpdateCommitIndexIfStale(leaderCommit int)
	String() string
}

// State describes the state of the raft algorithm, a server is in
type state struct {
	// latest term server has seen
	// (initialized to 0 on first boot, increases monotonically)
	currentTerm replog.Term

	// candidateId that received vote in current or (or nil if none)
	votedFor *string

	// log entries; each entry contains command for state machine,
	// and term when entry was received by leader (first index is 1)
	log replog.Log

	// index of highest log entry known to be committed (initialized to 0, increases monotonically)
	commitIndex int
}

// NewState returns a freshly initialized server state
func NewState() State {
	return &state{
		currentTerm: 0,
		votedFor:    nil,
		log:         replog.Log{},
		commitIndex: 0,
	}
}

// CurrentTerm returns the latest term the server has seen
func (s *state) CurrentTerm() replog.Term {
	return s.currentTerm
}

// IncrCurrentTerm increments the current term by one
func (s *state) IncrCurrentTerm(id *string) {
	s.currentTerm++
	s.votedFor = id
}

// UpdateTerm sets current term to the new term and resets votedFor
func (s *state) UpdateTerm(new replog.Term) {
	s.currentTerm = new
	s.votedFor = nil
}

func (s *state) VotedFor() *string {
	return s.votedFor
}

// CanVoteFor returns true if the server has not given a vote this term yet
// or it has given a vote to the requesting server
func (s *state) CanVoteFor(id string) bool {
	return s.votedFor == nil || *s.votedFor == id
}

func (s *state) SetVotedFor(id string) {
	s.votedFor = &id
}

func (s *state) Log() replog.Log {
	return s.log
}

func (s *state) DeleteConflictingAndAddNewEntries(prevLogIndex int, entries []replog.Entry) {
	l := s.log.DeleteConflictingEntries(prevLogIndex, entries)
	l = l.AppendEntries(prevLogIndex, entries)
	s.log = l
}

func (s *state) SetLog(l replog.Log) {
	s.log = l
}

func (s *state) CommitIndex() int {
	return s.commitIndex
}

func (s *state) UpdateCommitIndexIfStale(leaderCommit int) {
	if leaderCommit > s.commitIndex {
		lc := float64(leaderCommit)
		idx := float64(s.log.LastIndex())
		s.commitIndex = int(math.Min(lc, idx))
	}
}

// Equal returns true if x and y equal each other
func Equal(x, y State) bool {
	if x.CurrentTerm() != y.CurrentTerm() {
		return false
	}
	if x.VotedFor() != y.VotedFor() {
		return false
	}
	if !replog.Equal(x.Log(), y.Log()) {
		return false
	}
	return x.CommitIndex() == y.CommitIndex()
}

func makeString(s State) string {
	var vote string
	if s.VotedFor() == nil {
		vote = "<nil>"
	} else {
		vote = *s.VotedFor()
	}
	return fmt.Sprintf("State{%d: %s \t %d: %s}", s.CurrentTerm(), vote, s.CommitIndex(), s.Log().String())
}

func (s *state) String() string {
	return makeString(s)
}
