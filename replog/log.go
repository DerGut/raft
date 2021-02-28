package replog

import "fmt"

type Term int

type Entry struct {
	Cmd  string
	Term Term
}

type Log []Entry

func (l Log) DeleteConflictingEntries(prevLogIndex int, entries []Entry) Log {
	// Iterate over unsynced entries
	for i := prevLogIndex; i < l.LastIndex(); i++ {
		newEntryIndex := i - prevLogIndex
		if newEntryIndex >= len(entries) {
			return l
		}
		if l[i].Term != entries[newEntryIndex].Term {
			l = l[:i] // Delete this entry and all following
			break
		}
	}
	return l
}

func (l Log) Append(entry Entry) Log {
	return append(l, entry)
}

func (l Log) AppendEntries(prevLogIndex int, entries []Entry) Log {
	if len(entries) == 0 {
		return l
	}
	entryIndex := l.LastIndex() - prevLogIndex
	return append(l, entries[entryIndex:]...)
}

// (§5.3)
func (l Log) MatchesUntilNow(prevLogIndex int, prevLogTerm Term) bool {
	if l.LastIndex() < prevLogIndex {
		return false
	}
	if l.LastIndex() == 0 {
		if prevLogTerm == 0 {
			return true
		}
		return false
	}

	// prevLogIndex-1 for 1-based index correction
	return l[prevLogIndex-1].Term == prevLogTerm
}

func (l Log) IsMoreUpToDateThan(lastLogIndex int, lastLogTerm Term) bool {
	// Term is more recent
	if l.LastTerm() > lastLogTerm {
		return true
	}
	if l.LastTerm() < lastLogTerm {
		return false
	}

	// Contains more entries for current term
	if l.LastIndex() > lastLogIndex {
		return true
	}
	return false
}

func (l Log) LastIndex() int {
	// log index should start with 1
	return len(l)
}

func (l Log) LastTerm() Term {
	if len(l) == 0 {
		return 0
	}
	return l[len(l)-1].Term
}

func (l Log) At(index int) Entry {
	// Index should be at least 1
	return l[index-1]
}

func (l Log) Since(prevIndex int) []Entry {
	if prevIndex >= l.LastIndex() || prevIndex < 1 {
		return []Entry{}
	}

	return l[prevIndex:]
}

// Equal returns true if x and y equal each other
func Equal(x, y Log) bool {
	if len(x) != len(y) {
		return false
	}
	for i := 0; i < len(x); i++ {
		if x[i] != y[i] {
			return false
		}
	}

	return true
}

func (l Log) String() string {
	return fmt.Sprintf("Log[%d,%d]", l.LastIndex(), l.LastTerm())
}
