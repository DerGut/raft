package raft

import (
	"testing"

	"github.com/DerGut/kv-store/raft/rpc"
	"github.com/DerGut/kv-store/replog"
	"github.com/DerGut/kv-store/state"
)

func Test_processAppendEntries(t *testing.T) {
	type args struct {
		req rpc.AppendEntriesRequest
		s   state.State
	}
	tests := []struct {
		name      string
		args      args
		wantState state.State
		wantRes   rpc.AppendEntriesResponse
	}{
		{
			"Reply false if requesters term is < receivers term",
			args{
				rpc.AppendEntriesRequest{
					Term: 0,
				},
				state.NewTestState(1, nil, replog.Log{}, 0),
			},
			state.NewTestState(1, nil, replog.Log{}, 0),
			rpc.AppendEntriesResponse{Success: false, Term: 1},
		},
		{
			"Convert to Follower if requesters term > receivers term",
			args{
				rpc.AppendEntriesRequest{
					Term: 2,
				},
				state.NewTestState(1, nil, replog.Log{}, 0),
			},
			state.NewTestState(2, nil, replog.Log{}, 0),
			rpc.AppendEntriesResponse{Success: true, Term: 2},
		},
		{
			"Reply false if log doesn't contain an entry at prevLogIndex",
			args{
				rpc.AppendEntriesRequest{
					Term:         1,
					PrevLogIndex: 1,
				},
				state.NewTestState(1, nil, replog.Log{}, 0),
			},
			state.NewTestState(1, nil, replog.Log{}, 0),
			rpc.AppendEntriesResponse{Success: false, Term: 1},
		},
		{
			"Reply false if entry at prevLogIndex does not match prevLogTerm",
			args{
				rpc.AppendEntriesRequest{
					Term:         1,
					PrevLogIndex: 1,
					PrevLogTerm:  1,
				},
				state.NewTestState(1, nil, replog.Log{replog.Entry{Term: 0}}, 0),
			},
			state.NewTestState(1, nil, replog.Log{replog.Entry{Term: 0}}, 0),
			rpc.AppendEntriesResponse{Success: false, Term: 1},
		},
		{
			"Conflicting log entries get deleted and replaced with new ones",
			args{
				rpc.AppendEntriesRequest{
					PrevLogIndex: 1,
					Entries:      []replog.Entry{{Term: 1}, {Term: 1}},
				},
				state.NewTestState(0, nil, replog.Log{replog.Entry{}, replog.Entry{}, replog.Entry{}}, 0),
			},
			state.NewTestState(0, nil, replog.Log{replog.Entry{}, replog.Entry{Term: 1}, replog.Entry{Term: 1}}, 0),
			rpc.AppendEntriesResponse{Success: true, Term: 0},
		},
		{
			"Update commitIndex if leaderCommit is > current commitIndex",
			args{
				rpc.AppendEntriesRequest{
					PrevLogIndex: 0,
					PrevLogTerm:  0,
					LeaderCommit: 1,
					Entries:      []replog.Entry{{}},
				},
				state.NewTestState(0, nil, replog.Log{}, 0),
			},
			state.NewTestState(0, nil, replog.Log{replog.Entry{}}, 1),
			rpc.AppendEntriesResponse{Success: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRes := doProcessAppendEntries(tt.args.req, tt.args.s)
			if !state.Equal(tt.args.s, tt.wantState) {
				t.Errorf("doProcessAppendEntries() gotState = %v, want %v", tt.args.s, tt.wantState)
			}
			if gotRes != tt.wantRes {
				t.Errorf("doProcessAppendEntries() gotRes = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

func hasValue(c chan membership) bool {
	select {
	case <-c:
		return true
	default:
		return false
	}
}
