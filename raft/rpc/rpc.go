package rpc

import (
	"net"
	"net/rpc"
	"time"

	"github.com/DerGut/kv-store/replog"
)

const (
	timeout          = 1 * time.Second
	clusterRPCPath   = "/_raftRPC_/cluster"
	clusterDebugPath = "/debug/rpc/cluster"
	clientRPCPath    = "/_raftRPC_/client"
	ClientDebugPath  = "/debug/rpc/client"
)

type AppendEntriesRequest struct {
	replog.Term
	LeaderID     string
	PrevLogIndex int
	PrevLogTerm  replog.Term
	Entries      []replog.Entry
	LeaderCommit int
}

type AppendEntriesResponse struct {
	replog.Term
	Success bool
}

type RequestVoteRequest struct {
	replog.Term
	CandidateID  string
	LastLogIndex int
	LastLogTerm  replog.Term
}

type RequestVoteResponse struct {
	replog.Term
	VoteGranted bool
}

type ClientRequestRequest struct {
	Cmd string
}
type ClientRequestResponse struct {
	Success    bool
	LeaderAddr string
}

func registerClusterReceiver(rcvr *ClusterReceiver, address string) error {
	s := rpc.NewServer()
	if err := s.Register(rcvr); err != nil {
		return err
	}

	s.HandleHTTP(clusterRPCPath, clusterDebugPath)
	return nil
}

func registerClientReceiver(rcvr *ClientReceiver, address string) error {
	s := rpc.NewServer()
	if err := s.Register(rcvr); err != nil {
		return err
	}

	s.HandleHTTP(clientRPCPath, ClientDebugPath)
	return nil
}

func RegisterReceivers(cluster *ClusterReceiver, client *ClientReceiver, address string) (net.Listener, error) {
	if err := registerClusterReceiver(cluster, address); err != nil {
		return nil, err
	}
	if err := registerClientReceiver(client, address); err != nil {
		return nil, err
	}

	return net.Listen("tcp", address)
}
