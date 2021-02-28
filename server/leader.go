package server

import (
	"log"

	"github.com/DerGut/kv-store/raft/state"
	"github.com/DerGut/kv-store/replog"
)

func sendHeartBeat(options ClusterOptions, stateC chan state.State, resetC chan memberState) {
	state := <-stateC

	req := buildAppendEntriesRequest(state, options.Address, []replog.Entry{})
	cluster := NewCluster(options.Members)
	state = doSendHeartbeats(state, cluster, req, resetC)

	stateC <- state
}

func buildAppendEntriesRequest(state state.State, memberID string, entries []replog.Entry) AppendEntriesRequest {
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
