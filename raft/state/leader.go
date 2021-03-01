package state

import (
	"log"
	"sort"
)

type LeaderState struct {
	State
	NextIndex  map[string]int
	MatchIndex map[string]int
}

func NewLeaderStateFromState(state State, members []string) LeaderState {
	clusterSize := len(members)
	nextIndex := make(map[string]int, clusterSize)
	matchIndex := make(map[string]int, clusterSize)

	for _, member := range members {
		nextIndex[member] = state.Log().LastIndex() + 1
	}

	return LeaderState{state, nextIndex, matchIndex}
}

func (s *LeaderState) MajorityMatches() int {
	matchIndices := s.sortedMatchIndices()
	majorityIndex := len(matchIndices) / 2

	return matchIndices[majorityIndex]
}

func (s *LeaderState) sortedMatchIndices() []int {
	var matchIndices []int
	for _, matchIndex := range s.MatchIndex {
		matchIndices = append(matchIndices, matchIndex)
	}
	sort.Ints(matchIndices)

	return matchIndices
}

func (s *LeaderState) UpdateNextAndMatchingIndices(member string, success bool, lastIndex int) bool {
	if success {
		s.NextIndex[member] = lastIndex
		s.MatchIndex[member] = lastIndex
		return true
	}

	log.Println("Didn't succeed, decrementing nextIndex[", member, "]")
	if s.NextIndex[member] > 0 {
		s.NextIndex[member]--
	}

	return false
}
