package server

import (
	"log"

	"github.com/DerGut/raft/raft/state"
)

func sendHeartBeat(options ClusterOptions, stateC chan state.State, resetC chan memberState) {
	s := <-stateC

	req := buildAppendEntriesRequest(s, options.Address, []state.Entry{})
	cluster := NewCluster(options.Members)
	s = doSendHeartbeats(s, cluster, req, resetC)

	stateC <- s
}

func buildAppendEntriesRequest(state state.State, memberID string, entries []state.Entry) AppendEntriesRequest {
	l := state.Log()
	return AppendEntriesRequest{
		Term:         state.CurrentTerm(),
		LeaderID:     memberID,
		PrevLogIndex: l.LastIndex(),
		PrevLogTerm:  l.LastTerm(),
		Entries:      entries,
		LeaderCommit: state.CommitIndex(),
	}
}

func doSendHeartbeats(state state.State, cluster Cluster, req AppendEntriesRequest, resetC chan memberState) state.State {
	clusterSize := len(cluster)
	resC := make(chan *AppendEntriesResponse, clusterSize)
	errC := make(chan error, clusterSize)

	cluster.callAppendEntriesOnAll(&req, resC, errC)

	for i := 0; i < clusterSize; i++ {
		select {
		case res := <-resC:
			if isBehind(state, res.Term) {
				state.UpdateTerm(res.Term)
				log.Println("Discovered new term from heartbeat, reverting to FOLLOWER")
				resetC <- Follower
				return state
			}
		case <-errC:
			continue
		}
	}

	return state
}
