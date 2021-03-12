package server

import (
	"testing"

	"github.com/DerGut/raft/raft/state"
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
		wantRes         RequestVoteResponse
		wantStateChange bool
	}{
		{
			name: "Reply false and return new term if candidates term < receivers currentTerm",
			args: args{
				state.NewTestState(1, nil, state.Log{}, 0),
				RequestVoteRequest{Term: 0},
			},
			wantState:       state.NewTestState(1, nil, state.Log{}, 0),
			wantRes:         RequestVoteResponse{Term: 1, VoteGranted: false},
			wantStateChange: false,
		},
		{
			name: "Reply false if votedFor is set to another member",
			args: args{
				state.NewTestState(1, stringPtr("some member"), state.Log{}, 0),
				RequestVoteRequest{Term: 1, CandidateID: "another member"},
			},
			wantState:       state.NewTestState(1, stringPtr("some member"), state.Log{}, 0),
			wantRes:         RequestVoteResponse{1, false},
			wantStateChange: false,
		},
		{
			name: "Reply false if candidate log is shorter than receivers log",
			args: args{
				state.NewTestState(1, stringPtr("member"), state.Log{state.Entry{}}, 0),
				RequestVoteRequest{Term: 1, CandidateID: "member", LastLogIndex: 0},
			},
			wantState:       state.NewTestState(1, stringPtr("member"), state.Log{state.Entry{}}, 0),
			wantRes:         RequestVoteResponse{1, false},
			wantStateChange: false,
		},
		{
			name: "Reply false if candidate log is in older term than receivers log",
			args: args{
				state.NewTestState(1, stringPtr("member"), state.Log{state.Entry{Term: 1}}, 0),
				RequestVoteRequest{Term: 1, CandidateID: "member", LastLogIndex: 1, LastLogTerm: 0},
			},
			wantState:       state.NewTestState(1, stringPtr("member"), state.Log{state.Entry{Term: 1}}, 0),
			wantRes:         RequestVoteResponse{1, false},
			wantStateChange: false,
		},
		{
			name: "Reply true if candidate log is as up-to-date as receivers log",
			args: args{
				state.NewTestState(1, nil, state.Log{}, 0),
				RequestVoteRequest{Term: 1, CandidateID: "member"},
			},
			wantState:       state.NewTestState(1, stringPtr("member"), state.Log{}, 0),
			wantRes:         RequestVoteResponse{1, true},
			wantStateChange: true,
		},
		{
			name: "Update term and return to follower if receiver is behind candidate",
			args: args{
				state.NewTestState(1, nil, state.Log{}, 0),
				RequestVoteRequest{Term: 2, CandidateID: "member"},
			},
			wantState:       state.NewTestState(2, stringPtr("member"), state.Log{}, 0),
			wantRes:         RequestVoteResponse{2, true},
			wantStateChange: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stateC := make(chan memberState, 2) // TODO: Look at this
			gotState, gotRes := processRequestVote(tt.args.s, tt.args.req, stateC)
			gotStateChange := hasValue(stateC)
			if gotStateChange != tt.wantStateChange {
				t.Errorf("processRequestVote() gotStateChange = %t, want %t", gotStateChange, tt.wantStateChange)
			}
			if !state.Equal(gotState, tt.wantState) {
				t.Errorf("processRequestVote() gotState = %v, want %v", gotState, tt.wantState)
			}
			if gotRes != tt.wantRes {
				t.Errorf("processRequestVote() gotRes = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}

func Test_processAppendEntries(t *testing.T) {
	type args struct {
		s   state.State
		req AppendEntriesRequest
	}
	tests := []struct {
		name            string
		args            args
		wantState       state.State
		wantRes         AppendEntriesResponse
		wantStateChange bool
	}{
		{
			"Reply false if requesters term is < receivers term",
			args{
				state.NewTestState(1, nil, state.Log{}, 0),
				AppendEntriesRequest{
					Term: 0,
				},
			},
			state.NewTestState(1, nil, state.Log{}, 0),
			AppendEntriesResponse{Success: false, Term: 1},
			false,
		},
		{
			"Convert to Follower if requesters term > receivers term",
			args{
				state.NewTestState(1, nil, state.Log{}, 0),
				AppendEntriesRequest{
					Term: 2,
				},
			},
			state.NewTestState(2, nil, state.Log{}, 0),
			AppendEntriesResponse{Success: true, Term: 2},
			true,
		},
		{
			"Reply false if log doesn't contain an entry at prevLogIndex",
			args{
				state.NewTestState(1, nil, state.Log{}, 0),
				AppendEntriesRequest{
					Term:         1,
					PrevLogIndex: 1,
				},
			},
			state.NewTestState(1, nil, state.Log{}, 0),
			AppendEntriesResponse{Success: false, Term: 1},
			false,
		},
		{
			"Reply false if entry at prevLogIndex does not match prevLogTerm",
			args{
				state.NewTestState(1, nil, state.Log{state.Entry{Term: 0}}, 0),
				AppendEntriesRequest{
					Term:         1,
					PrevLogIndex: 1,
					PrevLogTerm:  1,
				},
			},
			state.NewTestState(1, nil, state.Log{state.Entry{Term: 0}}, 0),
			AppendEntriesResponse{Success: false, Term: 1},
			false,
		},
		{
			"Conflicting log entries get deleted and replaced with new ones",
			args{
				state.NewTestState(0, nil, state.Log{state.Entry{}, state.Entry{}, state.Entry{}}, 0),
				AppendEntriesRequest{
					PrevLogIndex: 1,
					Entries:      []state.Entry{{Term: 1}, {Term: 1}},
				},
			},
			state.NewTestState(0, nil, state.Log{state.Entry{}, state.Entry{Term: 1}, state.Entry{Term: 1}}, 0),
			AppendEntriesResponse{Success: true, Term: 0},
			false,
		},
		{
			"Update commitIndex if leaderCommit is > current commitIndex",
			args{
				state.NewTestState(0, nil, state.Log{}, 0),
				AppendEntriesRequest{
					PrevLogIndex: 0,
					PrevLogTerm:  0,
					LeaderCommit: 1,
					Entries:      []state.Entry{{}},
				},
			},
			state.NewTestState(0, nil, state.Log{state.Entry{}}, 1),
			AppendEntriesResponse{Success: true},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stateC := make(chan memberState, 1)
			gotState, gotRes := processAppendEntries(tt.args.s, tt.args.req, stateC)
			gotStateChange := hasValue(stateC)

			if gotStateChange != tt.wantStateChange {
				t.Errorf("processAppendEntries() gotStateChange = %t, want %t", gotStateChange, tt.wantStateChange)
			}
			if !state.Equal(gotState, tt.wantState) {
				t.Errorf("processAppendEntries() gotState = %v, want %v", gotState, tt.wantState)
			}
			if gotRes != tt.wantRes {
				t.Errorf("processAppendEntries() gotRes = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

func hasValue(c chan memberState) bool {
	select {
	case <-c:
		return true
	default:
		return false
	}
}
