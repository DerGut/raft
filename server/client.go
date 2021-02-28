package server

import (
	"context"
	"log"
	"net/rpc"
)

// TODO: Improve reconnect logic

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
		if client != nil {
			client.close()
		}
	}
}

// TODO: Add retry and context
// 		so that the client can continously retry until the caller cancels the context
//		when the election timer/ heartbeat timer has fired

func (c *Cluster) callRequestVoteOnAll(ctx context.Context, req *RequestVoteRequest, resC chan *RequestVoteResponse) {
	for memberID, client := range *c {
		log.Println("Calling requestVote on", memberID)
		var err error
		if client == nil {
			client, err = newClient(memberID)
			if err != nil {
				resC <- nil
				continue
			}
			(*c)[memberID] = client
		}
		go client.callRequestVote(ctx, req, resC)
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

func (c *client) callRequestVote(ctx context.Context, req *RequestVoteRequest, resCh chan *RequestVoteResponse) {
	res := RequestVoteResponse{}
	log.Println("Really calling requestVote on")

	for {
		errCh := make(chan error, 1)
		go func(ch chan error) {
			log.Println("snding request")
			ch <- c.rpcClient.Call("Server.RequestVote", req, &res)
			log.Println("received somehting", res)
		}(errCh)

		select {
		case err := <-errCh:
			if err != nil {
				log.Println("error here", err)
				continue
			} else {
				log.Println("response there", res)
				resCh <- &res
				return
			}
		case <-ctx.Done():
			log.Println("context canceled")
			resCh <- nil
			return
		}
	}

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
