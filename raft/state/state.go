package state

import (
	"fmt"
)

type State interface {
	CurrentTerm() Term
	IncrCurrentTerm()
	UpdateTerm(new Term)
	VotedFor() *string
	CanVoteFor(id string) bool
	SetVotedFor(id string)
	Log() Log
	DeleteConflictingAndAddNewEntries(prevLogIndex int, entries []Entry)
	SetLog(l Log)
	AppendToLog([]string)
	CommitIndex() int
	LeaderCommit(int)
	FollowerCommit(int)
	String() string
}

// State describes the state of the raft algorithm, a server is in
type state struct {
	Machine

	// latest term server has seen
	// (initialized to 0 on first boot, increases monotonically)
	currentTerm Term

	// candidateId that received vote in current or (or nil if none)
	votedFor *string

	// log entries; each entry contains command for state machine,
	// and term when entry was received by leader (first index is 1)
	log Log

	// index of highest log entry known to be committed (initialized to 0, increases monotonically)
	commitIndex int
}

// NewState returns a freshly initialized server state
func NewState(m Machine) State {
	return &state{
		Machine:     m,
		currentTerm: 0,
		votedFor:    nil,
		log:         Log{},
		commitIndex: 0,
	}
}

func NewTestState(term Term, votedFor *string, log Log, commitIndex int) State {
	return &state{nil, term, votedFor, log, commitIndex}
}

// CurrentTerm returns the latest term the server has seen
func (s *state) CurrentTerm() Term {
	return s.currentTerm
}

// IncrCurrentTerm increments the current term by one
func (s *state) IncrCurrentTerm() {
	s.currentTerm++
}

// UpdateTerm sets current term to the new term and resets votedFor
func (s *state) UpdateTerm(new Term) {
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

func (s *state) Log() Log {
	return s.log
}

func (s *state) DeleteConflictingAndAddNewEntries(prevLogIndex int, entries []Entry) {
	l := s.log.DeleteConflictingEntries(prevLogIndex, entries)
	l = l.AppendEntries(prevLogIndex, entries)
	s.log = l
}

func (s *state) SetLog(l Log) {
	s.log = l
}

func (s *state) AppendToLog(cmds []string) {
	entries := make([]Entry, len(cmds))
	for i, cmd := range cmds {
		entries[i] = Entry{Cmd: cmd, Term: s.CurrentTerm()}
	}
	l := s.Log().Append(entries)
	s.SetLog(l)
}

func (s *state) CommitIndex() int {
	return s.commitIndex
}

func (s *state) setCommitIndex(index int) {
	s.commitIndex = index
}

func (s *state) LeaderCommit(majorityMatch int) {
	new := s.highestMajorityMatch(majorityMatch)
	if new > s.CommitIndex() {
		s.commit(new)
	}
}

func (s *state) FollowerCommit(leaderCommit int) {
	new := s.leaderCommitOrLogLength(leaderCommit)
	s.commit(new)
}

func (s *state) commit(newCommitIndex int) {
	toCommit := s.log.Between(s.commitIndex, newCommitIndex)
	s.Machine.Commit(EntriesToCommands(toCommit))
	s.setCommitIndex(newCommitIndex)
}

func (s *state) leaderCommitOrLogLength(leaderCommit int) int {
	if leaderCommit > s.log.LastIndex() {
		return s.log.LastIndex()
	}
	return leaderCommit
}

func (s *state) highestMajorityMatch(majorityMatch int) int {
	for i := majorityMatch; i > s.CommitIndex(); i-- {
		if i > s.Log().LastIndex() {
			continue
		}
		if s.log.TermAt(i) == s.CurrentTerm() {
			return i
		}
	}

	return s.CommitIndex()
}

// Equal returns true if x and y equal each other
func Equal(x, y State) bool {
	if x.CurrentTerm() != y.CurrentTerm() {
		return false
	}
	if x.VotedFor() != nil && y.VotedFor() != nil {
		if *x.VotedFor() != *y.VotedFor() {
			return false
		}
	} else if x.VotedFor() != y.VotedFor() {
		return false
	}
	if !LogsEqual(x.Log(), y.Log()) {
		return false
	}
	return x.CommitIndex() == y.CommitIndex()
}

func (s *state) String() string {
	var vote string
	if s.VotedFor() == nil {
		vote = "<nil>"
	} else {
		vote = *s.VotedFor()
	}
	return fmt.Sprintf("State{%d: %s \t %d: %s}", s.CurrentTerm(), vote, s.CommitIndex(), s.Log().String())
}

func EntriesToCommands(entries []Entry) []string {
	cmds := make([]string, len(entries))
	for i, e := range entries {
		cmds[i] = e.Cmd
	}
	return cmds
}
