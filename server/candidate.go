package server

import (
	"context"
	"errors"
	"log"

	"github.com/DerGut/kv-store/raft/state"
)

func startElection(ctx context.Context, options ClusterOptions, stateC chan state.State, resetC chan memberState) {
	log.Println("Trying to start election, waiting for state")
	state := <-stateC
	log.Println("Aquired state")

	state = initializeNewTerm(state, options.Address, resetC)
	req := buildRequestVoteRequest(state, options.Address)

	log.Println("Starting new election")
	cluster := NewCluster(options.Members)
	state = runElection(ctx, state, cluster, req, resetC)

	stateC <- state
}

func initializeNewTerm(state state.State, memberID string, resetC chan memberState) state.State {
	state.IncrCurrentTerm()
	log.Println("Entering term", state.CurrentTerm())
	state.SetVotedFor(memberID)
	resetC <- Candidate

	return state
}

func buildRequestVoteRequest(state state.State, memberID string) RequestVoteRequest {
	l := state.Log()
	return RequestVoteRequest{
		Term:         state.CurrentTerm(),
		CandidateID:  memberID,
		LastLogIndex: l.LastIndex(),
		LastLogTerm:  l.LastTerm(),
	}
}

func runElection(ctx context.Context, state state.State, cluster Cluster, req RequestVoteRequest, resetC chan memberState) state.State {
	clusterSize := len(cluster) + 1
	resC := make(chan *RequestVoteResponse, clusterSize-1)

	log.Println("Calling request vote on all")
	go cluster.callRequestVoteOnAll(ctx, &req, resC)
	state, votes, err := countVotes(state, resC)
	if err != nil {
		log.Println("Discovered new term from votes, reverting to FOLLOWER")
		resetC <- Follower
		return state
	}

	log.Println("Received ", votes, " votes")
	if isMajority(votes+1, clusterSize) {
		log.Println("Is now LEADER")
		resetC <- Leader
		state = appendNoOpEntry(state)
	}
	return state
}

func countVotes(state state.State, resC chan *RequestVoteResponse) (state.State, int, error) {
	votesGranted := 0
	for i := 0; i < cap(resC); i++ {
		res := <-resC
		if res == nil {
			continue
		}
		if isBehind(state, res.Term) {
			state.UpdateTerm(res.Term)
			return state, 0, errors.New("Term was behind")
		}
		if res.VoteGranted {
			votesGranted++
		}
	}

	return state, votesGranted, nil
}

func isMajority(votes, clusterSize int) bool {
	if votes == 0 {
		return false
	}
	ratio := float64(votes) / float64(clusterSize)
	return ratio > 0.5
}

func appendNoOpEntry(s state.State) state.State {
	e := []state.Entry{{Term: s.CurrentTerm(), Cmd: ""}}
	l := s.Log().Append(e)
	s.SetLog(l)
	return s
}
