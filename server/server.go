package server

import (
	"log"
	"math"
	"time"

	"github.com/DerGut/kv-store/replog"
	"github.com/DerGut/kv-store/state"
	"github.com/DerGut/kv-store/timer"
)

type memberState int
type timeout int

const (
	Follower memberState = iota
	Candidate
	Leader

	election timeout = iota
	heartbeat
)

// Server is currently responsible for serving rpcs, handling the election,
// handling state changes, and operating on the log
type Server struct {
	options ClusterOptions

	timeout chan timeout
	timer   *time.Timer
	reset   chan memberState
	state   chan state.State
}

func (s memberState) String() string {
	return map[memberState]string{
		Follower:  "FOLLOWER",
		Candidate: "CANDIDATE",
		Leader:    "LEADER",
	}[s]
}

func NewServer(options ClusterOptions) *Server {
	return &Server{
		options: options,
		timeout: make(chan timeout, 100),
		reset:   make(chan memberState, 100),
		state:   make(chan state.State),
	}
}

// Run makes the server start listening to incoming RPC requests
func (s *Server) Run(debug bool) {
	log.Printf("Starting server %s of cluster %s\n", s.options.Address, s.options.Members)
	log.Println("Starts as FOLLOWER")
	s.timer = s.electionTimer()
	go func() {
		s.state <- state.NewState()
	}()
	for {
		select {
		case t := <-s.timeout:
			switch t {
			case election:
				log.Println("Is now CANDIDATE")
				go startElection(s.options, s.state, s.reset)
			case heartbeat:
				go sendHeartBeat(s.options, s.state, s.reset)
			}
		case r := <-s.reset:
			switch r {
			case Follower:
				log.Println("Is now FOLLOWER")
				fallthrough
			case Candidate:
				s.timer = s.electionTimer()
			case Leader:
				s.timer = s.heartbeatTimer()
			}
		}
	}
}

func (s *Server) electionTimer() *time.Timer {
	if s.timer != nil {
		s.timer.Stop()
	}
	t := timer.RandomElectionTimeout()
	return time.AfterFunc(t, func() {
		s.reset <- Candidate
		s.timeout <- election
	})
}

func (s *Server) heartbeatTimer() *time.Timer {
	s.timer.Stop()
	return time.AfterFunc(timer.HeartbeatTimeout, func() {
		s.reset <- Leader
		s.timeout <- heartbeat
	})
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
	// s.reset <- Follower // TODO: maybe only reset after verifying leader is valid

	state := <-s.state
	state, *res = processAppendEntries(state, req, s.reset)
	s.state <- state

	return nil
}

func processAppendEntries(s state.State, req AppendEntriesRequest, resetC chan memberState) (new state.State, res AppendEntriesResponse) {
	res.Term = responseTerm(req.Term, s.CurrentTerm())

	if isAhead(s, req.Term) {
		res.Success = false
		return s, res
	}
	if isBehind(s, req.Term) {
		log.Println("Entering new term", req.Term)
		s.UpdateTerm(req.Term)
		resetC <- Follower
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
	state := <-s.state
	state, *res = processRequestVote(state, req, s.reset)
	s.state <- state

	if res.VoteGranted {
		s.reset <- Follower
	}
	return nil
}

// TODO: See whether timerReset and revertToFollower can really be used together that many times

func processRequestVote(s state.State, req RequestVoteRequest, resetC chan memberState) (new state.State, res RequestVoteResponse) {
	res.Term = responseTerm(req.Term, s.CurrentTerm())

	if isAhead(s, req.Term) {
		res.VoteGranted = false
		return s, res
	}
	if isBehind(s, req.Term) {
		log.Println("Entering new term", req.Term)
		s.UpdateTerm(req.Term)
		resetC <- Follower
	}

	if s.CanVoteFor(req.CandidateID) {
		if !s.Log().IsMoreUpToDateThan(req.LastLogIndex, req.LastLogTerm) {
			resetC <- Follower
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
