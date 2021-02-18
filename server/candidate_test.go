package server

import "testing"

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
