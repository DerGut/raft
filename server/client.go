package server

import (
	"net/rpc"
)

type client struct {
	rpcClient *rpc.Client
}

func newClient(memberID string) (*client, error) {
	c, err := rpc.DialHTTP("tcp", string(memberID))
	if err != nil {
		return nil, err
	}
	return &client{rpcClient: c}, nil
}

func (c *client) close() error {
	return c.rpcClient.Close()
}

func (c *client) callRequestVote(req *RequestVoteRequest, resC chan *RequestVoteResponse, errC chan error) {
	res := RequestVoteResponse{}
	err := c.rpcClient.Call("Server.RequestVote", req, &res)
	if err != nil {
		errC <- err
		return
	}
	resC <- &res
}

func (c *client) callAppendEntries(req *AppendEntriesRequest, resC chan *AppendEntriesResponse, errC chan error) {
	res := AppendEntriesResponse{}
	err := c.rpcClient.Call("Server.AppendEntries", req, &res)
	if err != nil {
		errC <- err
		return
	}
	resC <- &res
}

type Cluster map[string]*client

func NewCluster(members []string) Cluster {
	c := make(Cluster, len(members))
	for _, memberID := range members {
		client, _ := newClient(memberID)
		c[memberID] = client
	}

	return c
}

func (c *Cluster) close() {
	for _, client := range *c {
		client.close()
	}
}

func (c *Cluster) callRequestVoteOnAll(req *RequestVoteRequest, resC chan *RequestVoteResponse, errC chan error) {
	for memberID, client := range *c {
		var err error
		if client == nil {
			client, err = newClient(memberID)
			if err != nil {
				errC <- err
				continue
			}
			(*c)[memberID] = client
		}
		go client.callRequestVote(req, resC, errC)
	}
}

func (c *Cluster) callAppendEntriesOnAll(req *AppendEntriesRequest, resC chan *AppendEntriesResponse, errC chan error) {
	for memberID, client := range *c {
		var err error
		if client == nil {
			client, err = newClient(memberID)
			if err != nil {
				errC <- err
				continue
			}
			(*c)[memberID] = client
		}
		go client.callAppendEntries(req, resC, errC)
	}
}
