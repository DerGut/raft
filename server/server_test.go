package server

import (
	"testing"

	"github.com/DerGut/kv-store/state"
	"github.com/DerGut/kv-store/timer"
)

func Test_RequestVotes(t *testing.T) {
	type args struct {
		req   RequestVoteRequest
		state state.State
	}
	type want struct {
		res   RequestVoteResponse
		state state.State
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"",
			args{
				RequestVoteRequest{
					Term:         0,
					CandidateID:  "127.0.0.1:3001",
					LastLogIndex: 0,
					LastLogTerm:  0,
				},
				&state.DummyState{},
			},
			want{
				RequestVoteResponse{
					Term:        0,
					VoteGranted: false,
				},
				&state.DummyState{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := buildTestServer(tt.args.state)
			var res RequestVoteResponse
			if err := s.RequestVote(tt.args.req, &res); err != nil {
				t.Errorf("Should not return an error %v", err)
			}
			if res != tt.want.res {
				t.Errorf("\ngot result =\n %v,\nwant\n %v", res, tt.want.res)
			}
			if !state.Equal(s.state, tt.want.state) {
				t.Errorf("\ngot state =\n %s,\nwant\n %s", s.state.String(), tt.want.state.String())
			}
		})
	}
}

func buildTestServer(state state.State) *Server {
	return &Server{
		MemberID:      "127.0.0.1:3000",
		stateChange:   make(chan memberState, 1),
		state:         state,
		electionTimer: timer.NewDummyElectionTimer(),
	}
}

func Test_isMajority(t *testing.T) {
	type args struct {
		votes       int
		clusterSize int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"No majority if total is 0", args{0, 0}, false},
		{"No majority if no votes", args{0, 1}, false},
		{"Majority if 1/1", args{1, 1}, true},
		{"No majority if 1/2", args{1, 2}, false},
		{"Majority if 2/3", args{2, 3}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMajority(tt.args.votes, tt.args.clusterSize); got != tt.want {
				t.Errorf("isMajority() = %v, want %v", got, tt.want)
			}
		})
	}
}
