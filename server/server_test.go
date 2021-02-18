package server

import (
	"reflect"
	"testing"

	"github.com/DerGut/kv-store/replog"
	"github.com/DerGut/kv-store/state"
)

func Test_processRequestVote(t *testing.T) {
	type args struct {
		s      state.State
		req    RequestVoteRequest
		stateC chan memberState
	}
	tests := []struct {
		name      string
		args      args
		wantState state.State
		wantRes   *RequestVoteResponse
	}{
		{
			"Reply false if term < currentTerm",
			args{
				state.NewTestState(1, nil, replog.Log{}, 0),
				RequestVoteRequest{},
				make(chan memberState, 1),
			},
			state.NewTestState(1, nil, replog.Log{}, 0),
			&RequestVoteResponse{1, false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotState, gotRes := processRequestVote(tt.args.s, tt.args.req, tt.args.stateC)
			if !reflect.DeepEqual(gotState, tt.wantState) {
				t.Errorf("processRequestVote() gotState = %v, want %v", gotState, tt.wantState)
			}
			if !reflect.DeepEqual(gotRes, tt.wantRes) {
				t.Errorf("processRequestVote() gotRes = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}
