package server

import (
	"reflect"
	"testing"

	"github.com/DerGut/kv-store/replog"
	"github.com/DerGut/kv-store/state"
)

func Test_processRequestVote(t *testing.T) {
	type args struct {
		s   state.State
		req RequestVoteRequest
	}
	tests := []struct {
		name            string
		args            args
		wantState       state.State
		wantRes         *RequestVoteResponse
		wantStateChange bool
	}{
		{
			name: "Reply false and return new term if candidates term < receivers currentTerm",
			args: args{
				state.NewTestState(1, nil, replog.Log{}, 0),
				RequestVoteRequest{Term: 0},
			},
			wantState:       state.NewTestState(1, nil, replog.Log{}, 0),
			wantRes:         &RequestVoteResponse{Term: 1, VoteGranted: false},
			wantStateChange: false,
		},
		{
			name: "Reply false if votedFor is set to another member",
			args: args{
				state.NewTestState(1, stringPtr("some member"), replog.Log{}, 0),
				RequestVoteRequest{Term: 1, CandidateID: "another member"},
			},
			wantState:       state.NewTestState(1, stringPtr("some member"), replog.Log{}, 0),
			wantRes:         &RequestVoteResponse{1, false},
			wantStateChange: false,
		},
		{
			name: "Reply false if candidate log is shorter than receivers log",
			args: args{
				state.NewTestState(1, stringPtr("member"), replog.Log{replog.Entry{}}, 0),
				RequestVoteRequest{Term: 1, CandidateID: "member", LastLogIndex: 0},
			},
			wantState:       state.NewTestState(1, stringPtr("member"), replog.Log{replog.Entry{}}, 0),
			wantRes:         &RequestVoteResponse{1, false},
			wantStateChange: false,
		},
		{
			name: "Reply false if candidate log is in older term than receivers log",
			args: args{
				state.NewTestState(1, stringPtr("member"), replog.Log{replog.Entry{Term: 1}}, 0),
				RequestVoteRequest{Term: 1, CandidateID: "member", LastLogIndex: 1, LastLogTerm: 0},
			},
			wantState:       state.NewTestState(1, stringPtr("member"), replog.Log{replog.Entry{Term: 1}}, 0),
			wantRes:         &RequestVoteResponse{1, false},
			wantStateChange: false,
		},
		{
			name: "Reply true if candidate log is as up-to-date as receivers log",
			args: args{
				state.NewTestState(1, nil, replog.Log{}, 0),
				RequestVoteRequest{Term: 1, CandidateID: "member"},
			},
			wantState:       state.NewTestState(1, stringPtr("member"), replog.Log{}, 0),
			wantRes:         &RequestVoteResponse{1, true},
			wantStateChange: true,
		},
		{
			name: "Update term and return to follower if receiver is behind candidate",
			args: args{
				state.NewTestState(1, nil, replog.Log{}, 0),
				RequestVoteRequest{Term: 2, CandidateID: "member"},
			},
			wantState:       state.NewTestState(2, stringPtr("member"), replog.Log{}, 0),
			wantRes:         &RequestVoteResponse{2, true},
			wantStateChange: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stateC := make(chan memberState, 2) // TODO: RequestVote can order two state changes. Is this bad?
			gotState, gotRes := processRequestVote(tt.args.s, tt.args.req, stateC)
			var gotStateChange bool
			select {
			case <-stateC:
				gotStateChange = true
			default:
				gotStateChange = false
			}
			if gotStateChange != tt.wantStateChange {
				t.Errorf("processRequestVote() gotStateChange = %t, want %t", gotStateChange, tt.wantStateChange)
			}
			if !reflect.DeepEqual(gotState, tt.wantState) {
				t.Errorf("processRequestVote() gotState = %v, want %v", gotState, tt.wantState)
			}
			if !reflect.DeepEqual(gotRes, tt.wantRes) {
				t.Errorf("processRequestVote() gotRes = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
