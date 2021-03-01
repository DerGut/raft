package raft

import (
	"context"
	"log"
	"time"

	"github.com/DerGut/kv-store/raft/rpc"
	"github.com/DerGut/kv-store/raft/state"
	"github.com/DerGut/kv-store/server"
	"github.com/DerGut/kv-store/timer"
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
	server.ClusterOptions

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
			log.Println("Processing AppendEntries")
			r.AppendEntriesResponses <- r.processAppendEntries(cancel, req)
		case req := <-r.RequestVoteRequests:
			log.Println("Processing RequestVote")
			r.RequestVoteResponses <- r.processRequestVote(cancel, req)
		case <-r.electionTicker.C:
			log.Println("Running Election")
			r.runElection(ctx)
		case <-r.heartbeatTicker.C:
			log.Println("Sending Heartbeat")
			r.heartbeat(ctx)
		case req := <-r.ClientReceiver.ClientRequests:
			log.Println("Processing ClientRequest")
			r.ClientResponses <- r.processClientRequest(ctx, req)
		case <-time.Tick(10 * time.Second):
			log.Printf("Tock: %#v", r.State)
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
	res := doProcessAppendEntries(req, r.State)
	log.Println(res)
	if res.Success {
		log.Println("Received heartbeat, resetting timer")
		cancel()
		r.membership = follower
		r.electionTicker.Reset(timer.RandomElectionTimeout() * 50)
		r.currentLeader = &req.LeaderID
	}
	return res
}

func (r *Raft) processRequestVote(cancel context.CancelFunc, req rpc.RequestVoteRequest) rpc.RequestVoteResponse {
	res := doProcessRequestVote(req, r.State)
	log.Println(res)
	if res.VoteGranted {
		cancel()
		r.membership = follower
		r.electionTicker.Reset(timer.RandomElectionTimeout() * 50)
	}
	return res
}

func (r *Raft) runElection(ctx context.Context) {
	r.membership = candidate
	success := doRunElection(ctx, r.State, r.ClusterOptions)
	if success {
		log.Println("Got majority of votes, now LEADER")
		r.electionTicker.Stop()
		r.membership = leader
		r.State = state.NewLeaderStateFromState(r.State, r.Members)
		r.heartbeatTicker = time.NewTicker(heartbeatTimeout)
		log.Println("Before initial log append", r.State)
		r.State.AppendToLog([]string{""})
		log.Println("After initial log append", r.State)
		log.Println("Sent initial heartbeat")
		r.heartbeat(ctx)
		log.Println("After initial heartbeat", r.State)
	} else {
		log.Println("No majority, back to FOLLOWER")
		r.membership = follower
		r.electionTicker.Reset(timer.RandomElectionTimeout() * 50)
	}
}

func (r *Raft) heartbeat(ctx context.Context) {
	ok := doHeartbeat(ctx, r.State, r.ClusterOptions)
	log.Println("heartbeat was ok", ok)

	if !ok {
		r.membership = follower
		r.heartbeatTicker.Stop()
		r.electionTicker = time.NewTicker(timer.RandomElectionTimeout() * 50)
	}
}

func (r *Raft) processClientRequest(ctx context.Context, req rpc.ClientRequestRequest) rpc.ClientRequestResponse {
	var res rpc.ClientRequestResponse
	if r.membership == leader {
		res = r.applyCommand(ctx, req)
	} else {
		res = r.forwardToLeader(req)
	}
	return res
}

func (r *Raft) forwardToLeader(req rpc.ClientRequestRequest) rpc.ClientRequestResponse {
	if r.currentLeader != nil {
		log.Println("Forwarding client to leader", *r.currentLeader)
		return rpc.ClientRequestResponse{Success: false, LeaderAddr: *r.currentLeader}
	}

	log.Println("Cannot forward client to leader; no leader known")
	return rpc.ClientRequestResponse{}
}
