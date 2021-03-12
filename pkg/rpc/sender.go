package rpc

import (
	"net/rpc"
)

func RequestVote(req RequestVoteRequest, addr string) (res *RequestVoteResponse, err error) {
	client, err := clusterClient(addr)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	err = client.Call("ClusterReceiver.RequestVote", req, &res)
	return
}

func AppendEntries(req AppendEntriesRequest, addr string) (res *AppendEntriesResponse, err error) {
	client, err := clusterClient(addr)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	err = client.Call("ClusterReceiver.AppendEntries", req, &res)
	return
}

func clusterClient(addr string) (*rpc.Client, error) {
	return rpc.DialHTTPPath("tcp", addr, clusterRPCPath)
}

func ClientRequest(req ClientRequestRequest, addr string) (res *ClientRequestResponse, err error) {
	client, err := clientClient(addr)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	err = client.Call("ClientReceiver.ClientRequest", req, &res)
	return
}

func clientClient(addr string) (*rpc.Client, error) {
	return rpc.DialHTTPPath("tcp", addr, clientRPCPath)
}
