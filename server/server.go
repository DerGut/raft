package server

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/DerGut/kv-store/replog"
	"github.com/DerGut/kv-store/state"
	"github.com/DerGut/kv-store/timer"
)

// Server is currently responsible for serving rpcs, handling the election,
// handling state changes, and operating on the log
type Server struct {
	MemberID string
	Cluster
	currentState memberState
	stateChange  chan memberState
	timer        *time.Timer

	state state.State
}

func (s *Server) String() string {
	return fmt.Sprintf("%s[%s][%d]", s.MemberID, s.currentState, s.state.CurrentTerm())
}

type memberState int

const (
	Follower  memberState = 0
	Candidate memberState = 1
	Leader    memberState = 2
)

func (s memberState) String() string {
	return map[memberState]string{
		Follower:  "FOLLOWER",
		Candidate: "CANDIDATE",
		Leader:    "LEADER",
	}[s]
}

func NewServer(uri string, members []string) *Server {
	log.Printf("Starting server %s of cluster %s\n", uri, members)

	cluster := NewCluster(members)

	s := Server{
		MemberID:     uri,
		Cluster:      cluster,
		currentState: Follower,
		stateChange:  make(chan memberState, 1),

		state: state.NewState(),
	}

	return &s
}

// Run makes the server start listening to incoming RPC requests
func (s *Server) Run(debug bool) error {
	log.Printf("%s starts as FOLLOWER\n", s)
	s.stateChange <- Follower

	for {
		switch <-s.stateChange {
		case Follower:
			s.timer = s.electionTimer()
			s.setState(Follower)
			break
		case Candidate:
			s.timer = s.electionTimer()
			s.setState(Candidate)
			s.startElection()
			break
		case Leader:
			s.timer = s.heartbeatTimer()
			s.setState(Leader)
			break
		}
	}
}

func (s *Server) electionTimer() *time.Timer {
	return time.AfterFunc(timer.RandomElectionTimeout(), s.electionTimeoutFunc)
}

func (s *Server) electionTimeoutFunc() {
	s.stateChange <- Candidate
	time.AfterFunc(timer.RandomElectionTimeout(), s.electionTimeoutFunc)
}

func (s *Server) heartbeatTimer() *time.Timer {
	return time.AfterFunc(timer.HeartbeatTimeout, s.heartbeatTimeoutFunc)
}

func (s *Server) heartbeatTimeoutFunc() {
	s.sendHeartBeat()
	s.timer = time.AfterFunc(timer.HeartbeatTimeout, s.heartbeatTimeoutFunc)
}

func (s *Server) setState(newState memberState) {
	if s.currentState != newState {
		s.currentState = newState
		log.Println(s, "is now", newState)
	}
}

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
	success bool
}

// AppendEntries is invoked by leader to rexplicate log entries (§5.3), It is also used as heartbeat (§5.2).
func (s *Server) AppendEntries(req AppendEntriesRequest, res *AppendEntriesResponse) error {
	s.timer = s.electionTimer()

	res.Term = responseTerm(req.Term, s.state.CurrentTerm())
	ok := s.ensureTerm(req.Term)
	if !ok {
		res.success = false
		return nil
	}

	if !s.state.Log().MatchesUntilNow(req.PrevLogIndex, req.PrevLogTerm) {
		res.success = false
		return nil
	}

	s.state.DeleteConflictingAndAddNewEntries(req.PrevLogIndex, req.Entries)
	s.state.UpdateCommitIndexIfStale(req.LeaderCommit)

	res.success = true
	return nil
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

// RequestVote is invoked by candidates to gather votes (§5.2).
func (s *Server) RequestVote(req RequestVoteRequest, res *RequestVoteResponse) error {
	log.Printf("%s received RequestVote from %s\n", s, req.CandidateID)

	res.Term = responseTerm(req.Term, s.state.CurrentTerm())
	ok := s.ensureTerm(req.Term)
	if !ok {
		res.VoteGranted = false
		return nil
	}

	if s.state.CanVoteFor(req.CandidateID) {
		if !s.state.Log().IsMoreUpToDateThan(req.LastLogIndex, req.LastLogTerm) {
			s.voteFor(req.CandidateID, res)
			return nil
		}
	}

	res.VoteGranted = false
	return nil
}

func processRequestVote(s state.State, req RequestVoteRequest, reset chan bool) (new state.State, res *RequestVoteResponse) {
	res.Term = responseTerm(req.Term, s.CurrentTerm())
	requesterIsAhead := ensureCurrentTermOrReturnToFollower(s, req.Term)
	if !requesterIsAhead {
		res.VoteGranted = false
		return s, res
	}

	if s.CanVoteFor(req.CandidateID) {
		if !s.Log().IsMoreUpToDateThan(req.LastLogIndex, req.LastLogTerm) {
			voteFor(req.CandidateID, s, res)
			return s, res
		}
	}

	res.VoteGranted = false
	return new, res
}

func ensureCurrentTermOrReturnToFollower(s state.State, reqTerm replog.Term) bool {
	if isAhead(s, reqTerm) {
		return false
	}
	if isBehind(s, reqTerm) {
		log.Println("Entering new term", reqTerm)
		s.UpdateTerm(reqTerm)
		s.returnToFollower()
	}
	return true
}

// isAhead compares the currentTerm with the term received by an RPC.
// it returns true, if the receivers term is ahead of the senders, false if it is not. (§5.1)
func isAhead(s state.State, term replog.Term) bool {
	return s.CurrentTerm() > term
}

func isBehind(s state.State, term replog.Term) bool {
	return s.CurrentTerm() < term
}

func responseTerm(reqTerm, currentTerm replog.Term) replog.Term {
	return maxTerm(reqTerm, currentTerm)
}

func maxTerm(x, y replog.Term) replog.Term {
	max := math.Max(float64(x), float64(y))
	return replog.Term(max)
}

func (s *Server) returnToFollower(timer *timer.ResettingTimer, stateC chan memberState) {
	if s.timer != nil {
		s.timer.Stop()
	}
	s.stateChange <- Follower
}

func voteFor(id string, s state.State, res *RequestVoteResponse) {
	res.VoteGranted = true
	s.timer = s.electionTimer()
	s.SetVotedFor(id)
}
