package raft

import (
	"context"
	"log"

	"github.com/DerGut/kv-store/raft/rpc"
	"github.com/DerGut/kv-store/raft/state"
	"github.com/DerGut/kv-store/replog"
	"github.com/DerGut/kv-store/server"
)

func doHeartbeat(ctx context.Context, s state.State, options server.ClusterOptions) bool {
	return updateFollowers(ctx, s.(state.LeaderState), options)
}

func (r *Raft) applyCommand(ctx context.Context, req rpc.ClientRequestRequest) rpc.ClientRequestResponse {
	log.Println("Applying command", req)
	r.State.AppendToLog(req.Cmd)
	if ok := updateFollowers(ctx, r.State.(LeaderState), r.ClusterOptions); ok {
		return rpc.ClientRequestResponse{Success: true}
	}
	return rpc.ClientRequestResponse{Success: false}
}

type memberResponse struct {
	member string
	*rpc.AppendEntriesResponse
}

func updateFollowers(ctx context.Context, s state.LeaderState, options server.ClusterOptions) bool {
	resCh := make(chan memberResponse, len(s.NextIndex))

	log.Println("Updating followers")
	for member, nextIndex := range s.NextIndex {
		prevIndex := nextIndex - 1
		newEntries := s.Log().Since(prevIndex)
		req := buildAppendEntriesRequest(s, options.Address, prevIndex, newEntries)
		appendEntries(ctx, member, req, resCh)
	}

	return awaitFollowerResponses(ctx, s, options, resCh)
}

// TODO: Debug & Test this
func awaitFollowerResponses(ctx context.Context, s state.LeaderState, options server.ClusterOptions, resCh chan memberResponse) bool {
	log.Println("Awaiting responses")
	numSucceeded := 0
	for numSucceeded < len(options.Members) {
		if isMajority(numSucceeded+1, len(options.Members)+1) {
			s.UpdateCommitIndex(s.MajorityMatches())
			return true
		}
		select {
		case res := <-resCh:
			if res.AppendEntriesResponse == nil {
				continue
			}
			if isBehind(s.CurrentTerm(), res.Term) {
				s.UpdateTerm(res.Term)
				log.Println("Discovered new term from heartbeat")
				return false
			}
			ok := s.UpdateNextAndMatchingIndices(res.member, *res.AppendEntriesResponse, s.Log().LastIndex())
			if ok {
				numSucceeded++
				continue
			}
			// Retry for indefinitely
			// TODO: do this in its own goroutine as this function returns once a majority has been reached
			prevIndex := s.NextIndex[res.member] - 1
			newEntries := s.Log().Since(prevIndex)
			req := buildAppendEntriesRequest(s, options.Address, prevIndex, newEntries)
			appendEntries(ctx, res.member, req, resCh)
		case <-ctx.Done():
			return false
		}
	}
	return false
}

func appendEntries(ctx context.Context, member string, req rpc.AppendEntriesRequest, resCh chan memberResponse) {
	log.Printf("Sending %#v to %s\n", req, member)
	go func() {
		res, _ := rpc.AppendEntries(req, member)
		select {
		case resCh <- memberResponse{member, res}:
		case <-ctx.Done():
		}
	}()
}

func buildAppendEntriesRequest(state state.State, leaderID string, prevIndex int, entries []replog.Entry) rpc.AppendEntriesRequest {
	var prevLogTerm replog.Term
	if prevIndex == 0 {
		prevLogTerm = 0
	} else {
		prevLogTerm = state.Log().At(prevIndex).Term
	}
	return rpc.AppendEntriesRequest{
		Term:         state.CurrentTerm(),
		LeaderID:     leaderID,
		PrevLogIndex: prevIndex,
		PrevLogTerm:  prevLogTerm,
		Entries:      entries,
		LeaderCommit: state.CommitIndex(),
	}
}
