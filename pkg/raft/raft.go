package raft

import (
	"context"
	"time"

	"github.com/DerGut/raft/pkg/rpc"
	"github.com/DerGut/raft/pkg/state"
	"github.com/DerGut/raft/timer"
)

// TODO: use function
const electionTimeout = 10000 * time.Millisecond
const heartbeatTimeout = 500 * time.Millisecond

type membership int

const (
	follower membership = iota
	candidate
	leader
)

type Raft struct {
	membership
	currentLeader *string
	ClusterOptions

	state.State

	rpc.ClusterReceiver
	rpc.ClientReceiver

	electionTicker, heartbeatTicker *time.Ticker
}

func (r *Raft) Run() {
	r.electionTicker = time.NewTicker(timer.RandomElectionTimeout() * 50)
	r.heartbeatTicker = stoppedTicker(heartbeatTimeout)

	ctx, cancel := context.WithCancel(context.Background())

	for {
		select {
		case req := <-r.AppendEntriesRequests:
			Info.Println("Processing AppendEntries")
			r.AppendEntriesResponses <- r.processAppendEntries(cancel, req)
		case req := <-r.RequestVoteRequests:
			Info.Println("Processing RequestVote")
			r.RequestVoteResponses <- r.processRequestVote(cancel, req)
		case <-r.electionTicker.C:
			Info.Println("Running Election")
			r.runElection(ctx)
		case <-r.heartbeatTicker.C:
			Info.Println("Sending Heartbeat")
			r.heartbeat(ctx)
		case req := <-r.ClientReceiver.ClientRequests:
			Info.Println("Processing ClientRequest")
			r.ClientResponses <- r.processClientRequest(ctx, req)
		}
		// Leaves the old context in go routines but overwrites the cancelFunc
		// context.WithCancel(ctx) // cancels old routines too
		ctx, cancel = context.WithCancel(context.Background())
	}
}

func stoppedTicker(d time.Duration) *time.Ticker {
	t := time.NewTicker(d)
	t.Stop()
	return t
}

func (r *Raft) processAppendEntries(cancel context.CancelFunc, req rpc.AppendEntriesRequest) rpc.AppendEntriesResponse {
	defer r.State.Persist()
	res := doProcessAppendEntries(req, r.State)
	if res.Success {
		cancel()
		r.membership = follower
		r.electionTicker.Reset(timer.RandomElectionTimeout() * 50)
		r.currentLeader = &req.LeaderID
	}
	return res
}

func (r *Raft) processRequestVote(cancel context.CancelFunc, req rpc.RequestVoteRequest) rpc.RequestVoteResponse {
	defer r.State.Persist()
	res := doProcessRequestVote(req, r.State)
	if res.VoteGranted {
		cancel()
		r.membership = follower
		r.electionTicker.Reset(timer.RandomElectionTimeout() * 50)
	}
	return res
}

func (r *Raft) runElection(ctx context.Context) {
	defer r.State.Persist()

	r.membership = candidate
	success := doRunElection(ctx, r.State, r.ClusterOptions)
	if success {
		Info.Println("Got majority of votes, now LEADER")
		r.electionTicker.Stop()
		r.membership = leader
		r.State = state.NewLeaderStateFromState(r.State, r.Members)
		r.heartbeatTicker = time.NewTicker(heartbeatTimeout)
		Info.Println("Before initial log append", r.State)
		r.State.AppendToLog([]string{""})
		Info.Println("After initial log append", r.State)
		Info.Println("Sent initial heartbeat")
		r.heartbeat(ctx)
		Info.Println("After initial heartbeat", r.State)
	} else {
		Info.Println("No majority, back to FOLLOWER")
		r.membership = follower
		r.electionTicker.Reset(timer.RandomElectionTimeout() * 50)
	}
}

func (r *Raft) heartbeat(ctx context.Context) {
	defer r.State.Persist()
	ok := doHeartbeat(ctx, r.State, r.ClusterOptions)

	if !ok {
		r.membership = follower
		r.heartbeatTicker.Stop()
		r.electionTicker = time.NewTicker(timer.RandomElectionTimeout() * 50)
	}
}

func (r *Raft) processClientRequest(ctx context.Context, req rpc.ClientRequestRequest) rpc.ClientRequestResponse {
	defer r.State.Persist()
	var res rpc.ClientRequestResponse
	if r.membership == leader {
		res = r.applyCommand(ctx, req)
	} else {
		res = r.forwardToLeader(req)
	}
	r.State.Persist()
	return res
}

func (r *Raft) forwardToLeader(req rpc.ClientRequestRequest) rpc.ClientRequestResponse {
	if r.currentLeader != nil {
		Info.Println("Forwarding client to leader", *r.currentLeader)
		return rpc.ClientRequestResponse{Success: false, LeaderAddr: *r.currentLeader}
	}

	Info.Println("Cannot forward client to leader; no leader known")
	return rpc.ClientRequestResponse{}
}
