package state

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/DerGut/raft/pkg/app"
)

const (
	snapshotSubDir   = "snap"
	stateSubDir      = "state"
	snapshottingSize = 1024
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
	LeaderCommit(int) string
	FollowerCommit(int)
	String() string
	Persist()
	Snapshot()
}

// State describes the state of the raft algorithm, a server is in
type state struct {
	app.StateMachine

	snapshotPath string

	cleanCh chan struct{}
	durable Durable

	// index of highest log entry known to be committed (initialized to 0, increases monotonically)
	commitIndex int
}

// NewState returns a freshly initialized server state
func NewState(m app.StateMachine, dirpath string) (State, error) {
	snapPath := filepath.Join(dirpath, snapshotSubDir)
	durablePath := filepath.Join(dirpath, stateSubDir)
	d, err := NewDurable(durablePath)
	if err != nil {
		return nil, err
	}

	c := make(chan struct{})
	go d.Clean(c)
	return &state{
		snapshotPath: snapPath,
		StateMachine: m,
		cleanCh:      c,
		durable:      *d,
		commitIndex:  0,
	}, nil
}

func NewTestState(term Term, votedFor *string, log Log, commitIndex int) State {
	return &state{nil, "", nil, Durable{}, commitIndex}
}

// CurrentTerm returns the latest term the server has seen
func (s *state) CurrentTerm() Term {
	return s.durable.CurrentTerm
}

// IncrCurrentTerm increments the current term by one
func (s *state) IncrCurrentTerm() {
	s.durable.CurrentTerm++
}

// UpdateTerm sets current term to the new term and resets votedFor
func (s *state) UpdateTerm(new Term) {
	s.durable.CurrentTerm = new
	s.durable.VotedFor = nil
}

func (s *state) VotedFor() *string {
	return s.durable.VotedFor
}

// CanVoteFor returns true if the server has not given a vote this term yet
// or it has given a vote to the requesting server
func (s *state) CanVoteFor(id string) bool {
	return s.durable.VotedFor == nil || *s.durable.VotedFor == id
}

func (s *state) SetVotedFor(id string) {
	s.durable.VotedFor = &id
}

func (s *state) Log() Log {
	return s.durable.Log
}

func (s *state) DeleteConflictingAndAddNewEntries(prevLogIndex int, entries []Entry) {
	l := s.durable.Log.DeleteConflictingEntries(prevLogIndex, entries)
	l = l.AppendEntries(prevLogIndex, entries)
	s.durable.Log = l
}

func (s *state) SetLog(l Log) {
	s.durable.Log = l
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

func (s *state) LeaderCommit(majorityMatch int) string {
	new := s.highestMajorityMatch(majorityMatch)
	if new > s.CommitIndex() {
		return s.commit(new)
	}
	return ""
}

func (s *state) FollowerCommit(leaderCommit int) {
	new := s.leaderCommitOrLogLength(leaderCommit)
	s.commit(new)
}

func (s *state) commit(newCommitIndex int) string {
	toCommit := s.durable.Log.Between(s.commitIndex, newCommitIndex)
	ret := []string{}
	for _, entry := range toCommit {
		if !isNoopCmd(entry.Cmd) {
			ret = append(ret, s.StateMachine.Commit(entry.Cmd))
		}
	}
	s.setCommitIndex(newCommitIndex)
	return strings.Join(ret, " ")
}

func isNoopCmd(cmd string) bool {
	return cmd == ""
}

func (s *state) leaderCommitOrLogLength(leaderCommit int) int {
	if leaderCommit > s.durable.Log.LastIndex() {
		return s.durable.Log.LastIndex()
	}
	return leaderCommit
}

func (s *state) highestMajorityMatch(majorityMatch int) int {
	for i := majorityMatch; i > s.CommitIndex(); i-- {
		if i > s.Log().LastIndex() {
			continue
		}
		if s.durable.Log.TermAt(i) == s.CurrentTerm() {
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

func (s *state) Persist() {
	if err := s.durable.Write(); err != nil {
		log.Println("Failed to persist state", err)
	}
}

func (s *state) Snapshot() {
	b := s.StateMachine.Snapshot()
	snap := snapshot{
		LastIncludedIndex: s.commitIndex,
		LastIncludedTerm:  s.Log().TermAt(s.commitIndex),
		ApplicationState:  b,
	}
	if err := snap.Write(s.snapshotPath); err != nil {
		log.Println("Failed to write snapshot", err)
	}
	deleteOldSnapshots(s.snapshotPath)
}

func timeToSnapshot(l Log) bool {
	return l.Size() > snapshottingSize
}
