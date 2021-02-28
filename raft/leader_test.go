package raft

import "testing"

func TestLeaderState_majorityMatches(t *testing.T) {
	type fields struct {
		nextIndex  map[string]int
		matchIndex map[string]int
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			"All indices 0 matches 0",
			fields{nil, map[string]int{"1": 0, "2": 0}},
			0,
		},
		{
			"Cluster size 3 matches largest matchIndex",
			fields{nil, map[string]int{"1": 1, "2": 2}},
			2,
		},
		{
			"Cluster size 5 matches second largest matchIndex",
			fields{nil, map[string]int{"1": 0, "2": 0, "3": 1, "4": 1}},
			1,
		},

		{
			"",
			fields{nil, map[string]int{"1": 0, "2": 0, "3": 1, "4": 2}},
			1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &LeaderState{
				nextIndex:  tt.fields.nextIndex,
				matchIndex: tt.fields.matchIndex,
			}
			if got := s.majorityMatches(); got != tt.want {
				t.Errorf("LeaderState.majorityMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}
