package raft

import (
	"log"

	"github.com/DerGut/kv-store/raft/rpc"
	"github.com/DerGut/kv-store/raft/state"
)

func doProcessAppendEntries(req rpc.AppendEntriesRequest, s state.State) rpc.AppendEntriesResponse {
	validLeader := isValidLeader(req, s.CurrentTerm(), s.Log())
	if req.Term > s.CurrentTerm() {
		log.Println("Entering new term", req.Term)
		s.UpdateTerm(req.Term)
	}
	if validLeader {
		log.Println("leader is valid")
		s.DeleteConflictingAndAddNewEntries(req.PrevLogIndex, req.Entries)
		s.UpdateCommitIndexIfStale(req.LeaderCommit)
	} else {
		log.Println("Leader is not valid")
	}

	return rpc.AppendEntriesResponse{Success: validLeader, Term: s.CurrentTerm()}
}

func isValidLeader(req rpc.AppendEntriesRequest, currentTerm state.Term, l state.Log) bool {
	if isBehind(req.Term, currentTerm) {
		log.Println("req is behind", req.Term, currentTerm)
		return false
	}
	if l.MatchesUntilNow(req.PrevLogIndex, req.PrevLogTerm) {
		return true
	}
	log.Printf("log doesn't match until now req: %#v log: %#v", req, l)
	return false
}
