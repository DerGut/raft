package raft

import (
	"context"
	"log"

	"github.com/DerGut/raft/pkg/rpc"
	"github.com/DerGut/raft/pkg/state"
)

func doRunElection(ctx context.Context, state state.State, options ClusterOptions) bool {
	initializeNewTerm(state, options.Address)

	req := buildRequestVoteRequest(state, options.Address)

	votes, ok := requestVotes(ctx, state, req, options)
	if ok && isMajority(votes+1, len(options.Members)+1) {
		return true
	}

	return false
}

func initializeNewTerm(state state.State, memberID string) {
	state.IncrCurrentTerm()
	log.Println("Entering term", state.CurrentTerm())
	state.SetVotedFor(memberID)
}

func buildRequestVoteRequest(state state.State, memberID string) rpc.RequestVoteRequest {
	return rpc.RequestVoteRequest{
		Term:         state.CurrentTerm(),
		CandidateID:  memberID,
		LastLogIndex: state.Log().LastIndex(),
		LastLogTerm:  state.Log().LastTerm(),
	}
}

func requestVotes(ctx context.Context, state state.State, req rpc.RequestVoteRequest, options ClusterOptions) (votes int, ok bool) {
	clusterSize := len(options.Members)
	resCh := make(chan *rpc.RequestVoteResponse, clusterSize)

	for _, member := range options.Members {
		go func(m string) {
			res, _ := rpc.RequestVote(req, m)
			select {
			case resCh <- res:
			case <-ctx.Done():
			}
		}(member)
	}

	return countVotes(ctx, state, resCh)
}

func isMajority(votes, clusterSize int) bool {
	if votes == 0 {
		return false
	}
	ratio := float64(votes) / float64(clusterSize)
	return ratio > 0.5
}

func countVotes(ctx context.Context, state state.State, resCh chan *rpc.RequestVoteResponse) (int, bool) {
	votesGranted := 0
	for i := 0; i < cap(resCh); i++ {
		select {
		case res := <-resCh:
			if res == nil {
				continue
			}
			if isBehind(state.CurrentTerm(), res.Term) {
				state.UpdateTerm(res.Term)
				return 0, false
			}
			if res.VoteGranted {
				votesGranted++
			}
		case <-ctx.Done():
			return 0, false
		}
	}

	return votesGranted, true
}
