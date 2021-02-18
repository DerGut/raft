package server

import (
	"errors"
	"log"
)

func (s *Server) startElection() {
	s.initializeNewTerm()
	log.Println(s, "Starting new election")

	req := s.buildRequestVoteRequest()
	resC := make(chan *RequestVoteResponse, len(s.Cluster))
	errC := make(chan error, len(s.Cluster))
	s.Cluster.callRequestVoteOnAll(&req, resC, errC)
	votes, err := s.countVotes(resC, errC)
	if err != nil {
		resetTimerAndReturnToFollower(s.stateChange)
		return
	}

	log.Println(s, "Received ", votes, " votes")
	if isMajority(votes+1, len(s.Cluster)+1) {
		s.stateChange <- Leader
	}
}

func (s *Server) initializeNewTerm() {
	s.state.IncrCurrentTerm()
	s.state.SetVotedFor(s.MemberID)
}

func (s *Server) buildRequestVoteRequest() RequestVoteRequest {
	l := s.state.Log()
	return RequestVoteRequest{
		Term:         s.state.CurrentTerm(),
		CandidateID:  s.MemberID,
		LastLogIndex: l.LastIndex(),
		LastLogTerm:  l.LastTerm(),
	}
}

func (s *Server) countVotes(resC chan *RequestVoteResponse, errC chan error) (int, error) {
	votesGranted := 0
	for i := 0; i < cap(resC); i++ {
		select {
		case res := <-resC:
			if isBehind(s.state, res.Term) {
				s.state.UpdateTerm(res.Term)
				return 0, errors.New("Term was behind")
			}
			if res.VoteGranted {
				votesGranted++
			} else {
			}
		case <-errC:
			continue
		}
	}

	return votesGranted, nil
}

func isMajority(votes, clusterSize int) bool {
	if votes == 0 {
		return false
	}
	ratio := float64(votes) / float64(clusterSize)
	return ratio > 0.5
}
