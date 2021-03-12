package raft

import (
	"testing"

	"github.com/DerGut/raft/pkg/rpc"
	"github.com/DerGut/raft/pkg/state"
)

func Test_processRequestVote(t *testing.T) {
	type args struct {
		req rpc.RequestVoteRequest
		s   state.State
	}
	tests := []struct {
		name      string
		args      args
		wantState state.State
		wantRes   rpc.RequestVoteResponse
	}{
		{
			name: "Reply false and return new term if candidates term < receivers currentTerm",
			args: args{
				rpc.RequestVoteRequest{Term: 0},
				state.NewTestState(1, nil, state.Log{}, 0),
			},
			wantState: state.NewTestState(1, nil, state.Log{}, 0),
			wantRes:   rpc.RequestVoteResponse{Term: 1, VoteGranted: false},
		},
		{
			name: "Reply false if votedFor is set to another member",
			args: args{
				rpc.RequestVoteRequest{Term: 1, CandidateID: "another member"},
				state.NewTestState(1, stringPtr("some member"), state.Log{}, 0),
			},
			wantState: state.NewTestState(1, stringPtr("some member"), state.Log{}, 0),
			wantRes:   rpc.RequestVoteResponse{1, false},
		},
		{
			name: "Reply false if candidate log is shorter than receivers log",
			args: args{
				rpc.RequestVoteRequest{Term: 1, CandidateID: "member", LastLogIndex: 0},
				state.NewTestState(1, stringPtr("member"), state.Log{state.Entry{}}, 0),
			},
			wantState: state.NewTestState(1, stringPtr("member"), state.Log{state.Entry{}}, 0),
			wantRes:   rpc.RequestVoteResponse{1, false},
		},
		{
			name: "Reply false if candidate log is in older term than receivers log",
			args: args{
				rpc.RequestVoteRequest{Term: 1, CandidateID: "member", LastLogIndex: 1, LastLogTerm: 0},
				state.NewTestState(1, stringPtr("member"), state.Log{state.Entry{Term: 1}}, 0),
			},
			wantState: state.NewTestState(1, stringPtr("member"), state.Log{state.Entry{Term: 1}}, 0),
			wantRes:   rpc.RequestVoteResponse{1, false},
		},
		{
			name: "Reply true if candidate log is as up-to-date as receivers log",
			args: args{
				rpc.RequestVoteRequest{Term: 1, CandidateID: "member"},
				state.NewTestState(1, nil, state.Log{}, 0),
			},
			wantState: state.NewTestState(1, stringPtr("member"), state.Log{}, 0),
			wantRes:   rpc.RequestVoteResponse{1, true},
		},
		{
			name: "Update term and return to follower if receiver is behind candidate",
			args: args{
				rpc.RequestVoteRequest{Term: 2, CandidateID: "member"},
				state.NewTestState(1, nil, state.Log{}, 0),
			},
			wantState: state.NewTestState(2, stringPtr("member"), state.Log{}, 0),
			wantRes:   rpc.RequestVoteResponse{2, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRes := doProcessRequestVote(tt.args.req, tt.args.s)
			if !state.Equal(tt.args.s, tt.wantState) {
				t.Errorf("processRequestVote() gotState = %v, want %v", tt.args.s, tt.wantState)
			}
			if gotRes != tt.wantRes {
				t.Errorf("processRequestVote() gotRes = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}
