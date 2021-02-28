package rpc

import (
	"errors"
	"log"
	"time"
)

type ClusterReceiver struct {
	AppendEntriesRequests  chan AppendEntriesRequest
	AppendEntriesResponses chan AppendEntriesResponse
	RequestVoteRequests    chan RequestVoteRequest
	RequestVoteResponses   chan RequestVoteResponse
}

func (s *ClusterReceiver) AppendEntries(req AppendEntriesRequest, res *AppendEntriesResponse) error {
	s.AppendEntriesRequests <- req
	select {
	case *res = <-s.AppendEntriesResponses:
		return nil
	case <-time.After(timeout):
		return errors.New("AppendEntries timed out")
	}
}

func (s *ClusterReceiver) RequestVote(req RequestVoteRequest, res *RequestVoteResponse) error {
	s.RequestVoteRequests <- req
	log.Println("Received RequestVote", req)
	select {
	case r := <-s.RequestVoteResponses:
		*res = r
		log.Println("Responding", res)
		return nil
	case <-time.After(timeout):
		return errors.New("RequestVote timed out")
	}
}

type ClientReceiver struct {
	ClientRequests  chan ClientRequestRequest
	ClientResponses chan ClientRequestResponse
}

func (s *ClientReceiver) ClientRequest(req ClientRequestRequest, res *ClientRequestResponse) error {
	s.ClientRequests <- req
	select {
	case *res = <-s.ClientResponses:
		return nil
	case <-time.After(timeout):
		return errors.New("ClientRequest timed out")
	}
}
