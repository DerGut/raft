package raft

import (
	"log"

	"github.com/DerGut/kv-store/raft/rpc"
	"github.com/DerGut/kv-store/raft/state"
)

func doProcessRequestVote(req rpc.RequestVoteRequest, s state.State) (res rpc.RequestVoteResponse) {
	validCandidate := isValidCandidate(req, s.CurrentTerm())
	if req.Term > s.CurrentTerm() {
		log.Println("Entering new term", req.Term)
		s.UpdateTerm(req.Term)
	}

	if validCandidate && isAbleToVote(req, s) {
		s.SetVotedFor(req.CandidateID)
		return rpc.RequestVoteResponse{VoteGranted: true, Term: s.CurrentTerm()}
	}

	return rpc.RequestVoteResponse{VoteGranted: false, Term: s.CurrentTerm()}
}

func isValidCandidate(req rpc.RequestVoteRequest, currentTerm state.Term) bool {
	return !isBehind(req.Term, currentTerm)
}

func isAbleToVote(req rpc.RequestVoteRequest, s state.State) bool {
	if s.CanVoteFor(req.CandidateID) {
		if !s.Log().IsMoreUpToDateThan(req.LastLogIndex, req.LastLogTerm) {
			return true
		}
	}
	return false
}
