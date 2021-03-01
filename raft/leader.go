package raft

import (
	"context"
	"errors"
	"log"

	"github.com/DerGut/kv-store/raft/rpc"
	"github.com/DerGut/kv-store/raft/state"
	"github.com/DerGut/kv-store/server"
)

func doHeartbeat(ctx context.Context, s state.State, options server.ClusterOptions) bool {
	return updateFollowers(ctx, s.(state.LeaderState), options)
}

func (r *Raft) applyCommand(ctx context.Context, req rpc.ClientRequestRequest) rpc.ClientRequestResponse {
	log.Println("Applying command", req)
	r.State.AppendToLog(req.Cmds)
	if ok := updateFollowers(ctx, r.State.(state.LeaderState), r.ClusterOptions); ok {
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
	for _, member := range options.Members {
		req := buildAppendEntriesRequest(s, options.Address, member)
		appendEntries(ctx, member, req, resCh)
	}

	return awaitFollowerResponses(ctx, s, options, resCh)
}

// TODO: Debug & Test this
// Should return after majority has been reached but continue to run in the background until rest of the cluster has agreed.
// Should be cancellable during all of the operation.
func awaitFollowerResponses(ctx context.Context, s state.LeaderState, options server.ClusterOptions, resCh chan memberResponse) bool {
	log.Println("Awaiting responses")
	numAgreed := 1
	clusterSize := len(options.Members) + 1
	for !isMajority(numAgreed, clusterSize) {
		ok, err := awaitResponse(ctx, s, options.Address, resCh)
		if err != nil {
			return false
		}
		if ok {
			numAgreed++
		}
	}

	s.UpdateCommitIndex(s.MajorityMatches())

	// TODO: Retry for rest of cluster

	return true
}

func awaitResponse(ctx context.Context, s state.LeaderState, leaderID string, resCh chan memberResponse) (bool, error) {
	select {
	case res := <-resCh:
		if res.AppendEntriesResponse == nil {
			// Error, try again
			return false, nil
		}
		if isBehind(s.CurrentTerm(), res.Term) {
			s.UpdateTerm(res.Term)
			log.Println("Discovered new term from heartbeat")
			return false, errors.New("Revert to FOLLOWER")
		}
		if res.Success {
			// TODO: lastIndex could be overwritten already once this is running in a goroutine
			s.UpdateIndices(res.member, s.Log().LastIndex())
			return true, nil
		}

		s.DecrementNextIndex(res.member)
		req := buildAppendEntriesRequest(s, leaderID, res.member)
		appendEntries(ctx, res.member, req, resCh)
		return false, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
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

func buildAppendEntriesRequest(s state.LeaderState, leaderID string, member string) rpc.AppendEntriesRequest {
	prevIndex := s.NextIndex[member] - 1
	newEntries := s.Log().Since(prevIndex)

	return rpc.AppendEntriesRequest{
		Term:         s.CurrentTerm(),
		LeaderID:     leaderID,
		PrevLogIndex: prevIndex,
		PrevLogTerm:  s.Log().TermAt(prevIndex),
		Entries:      newEntries,
		LeaderCommit: s.CommitIndex(),
	}
}
