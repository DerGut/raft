package raft

import (
	"context"
	"reflect"
	"testing"

	"github.com/DerGut/raft/raft/rpc"
	"github.com/DerGut/raft/raft/state"
	"github.com/DerGut/raft/server"
)

func Test_doRunElection(t *testing.T) {
	t.Fail()
	type args struct {
		state   state.State
		options server.ClusterOptions
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := doRunElection(context.TODO(), tt.args.state, tt.args.options); got != tt.want {
				t.Errorf("doRunElection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_initializeNewTerm(t *testing.T) {
	t.Fail()
	type args struct {
		state    state.State
		memberID string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initializeNewTerm(tt.args.state, tt.args.memberID)
		})
	}
}

func Test_buildRequestVoteRequest(t *testing.T) {
	t.Fail()
	type args struct {
		state    state.State
		memberID string
	}
	tests := []struct {
		name string
		args args
		want rpc.RequestVoteRequest
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildRequestVoteRequest(tt.args.state, tt.args.memberID); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildRequestVoteRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_requestVotes(t *testing.T) {
	t.Fail()
	type args struct {
		state   state.State
		req     rpc.RequestVoteRequest
		options server.ClusterOptions
	}
	tests := []struct {
		name      string
		args      args
		wantVotes int
		wantOk    bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVotes, gotOk := requestVotes(context.TODO(), tt.args.state, tt.args.req, tt.args.options)
			if gotVotes != tt.wantVotes {
				t.Errorf("requestVotes() gotVotes = %v, want %v", gotVotes, tt.wantVotes)
			}
			if gotOk != tt.wantOk {
				t.Errorf("requestVotes() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func Test_isMajority(t *testing.T) {
	t.Fail()
	type args struct {
		votes       int
		clusterSize int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMajority(tt.args.votes, tt.args.clusterSize); got != tt.want {
				t.Errorf("isMajority() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_countVotes(t *testing.T) {
	t.Fail()
	type args struct {
		state state.State
		resCh chan *rpc.RequestVoteResponse
	}
	tests := []struct {
		name  string
		args  args
		want  int
		want1 bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := countVotes(context.TODO(), tt.args.state, tt.args.resCh)
			if got != tt.want {
				t.Errorf("countVotes() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("countVotes() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
