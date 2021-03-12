package state

import "testing"

func TestLog_DeleteConflictingEntries(t *testing.T) {
	type args struct {
		prevLogIndex int
		entries      []Entry
	}
	tests := []struct {
		name string
		l    Log
		args args
		want Log
	}{
		{"Empty log has no conflicting entries with another empty log", Log{}, args{0, []Entry{}}, Log{}},
		{"Empty log has no conflicting entries first append", Log{}, args{0, []Entry{{Term: 0}, {Term: 0}}}, Log{}},
		{"Empty append does not delete any entries", Log{Entry{Term: 0}}, args{0, []Entry{}}, Log{Entry{Term: 0}}},
		{"Append with same existing entry does not conflict", Log{Entry{Term: 0}}, args{0, []Entry{{Term: 0}}}, Log{Entry{Term: 0}}},
		{"Append with newer term at existing entry conflicts", Log{Entry{Term: 0}}, args{0, []Entry{{Term: 1}}}, Log{}},
		{"Append with newer term at existing entry conflicts with all after that", Log{Entry{Term: 0}, Entry{Term: 0}, Entry{Term: 0}}, args{0, []Entry{{Term: 0}, {Term: 1}, {Term: 1}}}, Log{Entry{Term: 0}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.l.DeleteConflictingEntries(tt.args.prevLogIndex, tt.args.entries)
			if !LogsEqual(got, tt.want) {
				t.Errorf("Log = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLog_AppendEntries(t *testing.T) {
	type args struct {
		prevLogIndex int
		entries      []Entry
	}
	tests := []struct {
		name string
		l    Log
		args args
		want Log
	}{
		{"Empty log stays empty with no new entries", Log{}, args{0, []Entry{}}, Log{}},
		{"Empty log gets new entry", Log{}, args{0, []Entry{{}}}, Log{Entry{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.l.AppendEntries(tt.args.prevLogIndex, tt.args.entries)
			if !LogsEqual(got, tt.want) {
				t.Errorf("got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLog_MatchesUntilNow(t *testing.T) {
	type args struct {
		prevLogIndex int
		prevLogTerm  Term
	}
	tests := []struct {
		name string
		l    Log
		args args
		want bool
	}{
		{"Empty log matches with empty log", Log{}, args{0, 0}, true},
		{"Empty log does not match with prevLogIndex greater than its own lastIndex", Log{}, args{1, 0}, false},
		{"Empty log does not match with same previous index and newer term", Log{}, args{0, 1}, false},
		{"Log matches with same previous index and lastIndex and same term", Log{{Term: 1}}, args{1, 1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.l.MatchesUntilNow(tt.args.prevLogIndex, tt.args.prevLogTerm); got != tt.want {
				t.Errorf("Log.MatchesUntilNow() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLog_IsMoreUpToDateThan(t *testing.T) {
	type args struct {
		lastLogIndex int
		lastLogTerm  Term
	}
	tests := []struct {
		name string
		l    *Log
		args args
		want bool
	}{
		{"Empty log is not more up-to-date than empty log with term 0", &Log{}, args{0, 0}, false},
		{"Empty log is not more up-to-date than empty log with term 1", &Log{}, args{0, 1}, false},
		{"Empty log is not more up-to-date than log of size 1 with term 0", &Log{}, args{1, 0}, false},
		{"Empty log is not more up-to-date than log of size 1 with term 1", &Log{}, args{1, 1}, false},
		{"Log with older term and shorter length is not more up-to-date", &Log{Entry{Term: 0}}, args{2, 1}, false},
		{"Log with older term and same length is not more up-to-date", &Log{Entry{Term: 0}}, args{1, 1}, false},
		{"Log with older term and greater length is not more up-to-date", &Log{Entry{Term: 0}, Entry{Term: 0}}, args{1, 1}, false},
		{"Log with same term and shorter length is not more up-to-date", &Log{Entry{Term: 1}}, args{2, 1}, false},
		{"Log with same term and same length is not more up-to-date", &Log{Entry{Term: 1}}, args{1, 1}, false},
		{"Log with same term and greater length is more up-to-date", &Log{Entry{Term: 1}, Entry{Term: 1}}, args{1, 1}, true},
		{"Log with newer term and shorter length is more up-to-date", &Log{Entry{Term: 2}}, args{2, 1}, true},
		{"Log with newer term and same length is more up-to-date", &Log{Entry{Term: 2}}, args{1, 1}, true},
		{"Log with newer term and greater length is more up-to-date", &Log{Entry{Term: 2}, Entry{Term: 2}}, args{1, 1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.l.IsMoreUpToDateThan(tt.args.lastLogIndex, tt.args.lastLogTerm); got != tt.want {
				t.Errorf("Log.IsMoreUpToDateThan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLog_LastIndex(t *testing.T) {
	tests := []struct {
		name string
		l    Log
		want int
	}{
		{"Empty log has index 0", Log{}, 0},
		{"Log of size 1 has index 1", Log{Entry{}}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.l.LastIndex(); got != tt.want {
				t.Errorf("Log.LastIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLog_LastIndex_AfterDelete(t *testing.T) {
	l := Log{Entry{}, Entry{}, Entry{}}
	l = l[:1]
	if l.LastIndex() != 1 {
		t.Fail()
	}
}

func TestLog_LastTerm(t *testing.T) {
	tests := []struct {
		name string
		l    Log
		want Term
	}{
		{"Empty log has term 0", Log{}, 0},
		{"Log has term of only entry", Log{Entry{Term: 1}}, 1},
		{"Log has term of last entry", Log{Entry{Term: 1}, Entry{Term: 2}}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.l.LastTerm(); got != tt.want {
				t.Errorf("Log.LastTerm() = %v, want %v", got, tt.want)
			}
		})
	}
}
