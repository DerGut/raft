package state

import "github.com/DerGut/kv-store/replog"

type DummyState struct {
	currentTerm replog.Term

	// candidateId that received vote in current or (or nil if none)
	votedFor *string

	// log entries; each entry contains command for state machine,
	// and term when entry was received by leader (first index is 1)
	log replog.Log

	// index of highest log entry known to be committed (initialized to 0, increases monotonically)
	commitIndex int
}

// func (s *DummyState) CurrentTerm() replog.Term
// func (s *DummyState) IncrCurrentTerm(id *string)
// func (s *DummyState) UpdateTerm(new replog.Term)
// func (s *DummyState) VotedFor() *string
// func (s *DummyState) CanVoteFor(id string) bool
// func (s *DummyState) SetVotedFor(id string)
// func (s *DummyState) Log() replog.Log
// func (s *DummyState) DeleteConflictingAndAddNewEntries(prevLogIndex int, entries []replog.Entry)
// func (s *DummyState) SetLog(l replog.Log)
// func (s *DummyState) CommitIndex() int
// func (s *DummyState) UpdateCommitIndexIfStale(leaderCommit int)

// func (s *DummyState) String() string {
// 	return makeString(s)
// }
