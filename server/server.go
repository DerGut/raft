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
	return fmt.Sprintf("%s[%d]:", s.currentState, s.state.CurrentTerm())
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

	return &Server{
		MemberID:     uri,
		Cluster:      NewCluster(members),
		currentState: Follower,
		stateChange:  make(chan memberState, 100),

		state: state.NewState(),
	}
}

// TODO: Decouple state setting from timer resetting

// Run makes the server start listening to incoming RPC requests
func (s *Server) Run(debug bool) {
	log.Printf("%s starts as FOLLOWER\n", s)
	s.timer = s.electionTimer()

	for {
		switch <-s.stateChange {
		case Follower:
			s.setState(Follower)
			break
		case Candidate:
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
	if s.timer != nil {
		s.timer.Stop()
	}
	t := timer.RandomElectionTimeout()
	return time.AfterFunc(t, s.electionTimeoutFunc)
}

func (s *Server) electionTimeoutFunc() {
	s.stateChange <- Candidate
	s.timer = s.electionTimer()
}

func (s *Server) heartbeatTimer() *time.Timer {
	s.timer.Stop()
	return time.AfterFunc(timer.HeartbeatTimeout, s.heartbeatTimeoutFunc)
}

func (s *Server) heartbeatTimeoutFunc() {
	s.sendHeartBeat()
	s.timer = s.heartbeatTimer()
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
	Success bool
}

// AppendEntries is invoked by leader to rexplicate log entries (§5.3), It is also used as heartbeat (§5.2).
func (s *Server) AppendEntries(req AppendEntriesRequest, res *AppendEntriesResponse) error {
	s.timer = s.electionTimer()

	s.state, *res = processAppendEntries(s.state, req, s.stateChange)

	return nil
}

func processAppendEntries(s state.State, req AppendEntriesRequest, stateC chan memberState) (new state.State, res AppendEntriesResponse) {
	res.Term = responseTerm(req.Term, s.CurrentTerm())

	if isAhead(s, req.Term) {
		res.Success = false
		return s, res
	}
	if isBehind(s, req.Term) {
		log.Println("Entering new term", req.Term)
		s.UpdateTerm(req.Term)
		returnToFollower(stateC)
	}

	if !s.Log().MatchesUntilNow(req.PrevLogIndex, req.PrevLogTerm) {
		res.Success = false
		return s, res
	}

	s.DeleteConflictingAndAddNewEntries(req.PrevLogIndex, req.Entries)
	s.UpdateCommitIndexIfStale(req.LeaderCommit)

	res.Success = true
	return s, res
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
	s.state, *res = processRequestVote(s.state, req, s.stateChange)
	if res.VoteGranted {
		s.timer = s.electionTimer()
	}
	return nil
}

// TODO: See whether timerReset and revertToFollower can really be used together that many times

func processRequestVote(s state.State, req RequestVoteRequest, stateC chan memberState) (new state.State, res RequestVoteResponse) {
	res.Term = responseTerm(req.Term, s.CurrentTerm())

	if isAhead(s, req.Term) {
		res.VoteGranted = false
		return s, res
	}
	if isBehind(s, req.Term) {
		log.Println("Entering new term", req.Term)
		s.UpdateTerm(req.Term)
		returnToFollower(stateC)
	}

	if s.CanVoteFor(req.CandidateID) {
		if !s.Log().IsMoreUpToDateThan(req.LastLogIndex, req.LastLogTerm) {
			returnToFollower(stateC)
			s.SetVotedFor(req.CandidateID)
			res.VoteGranted = true
			return s, res
		}
	}

	res.VoteGranted = false
	return s, res
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

func returnToFollower(stateC chan memberState) {
	stateC <- Follower
}
