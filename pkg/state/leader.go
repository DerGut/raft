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

func (s *LeaderState) UpdateIndices(member string, newIndex int) {
	s.NextIndex[member] = newIndex
	s.MatchIndex[member] = newIndex
}

func (s *LeaderState) DecrementNextIndex(member string) {
	log.Println("Didn't succeed, decrementing nextIndex[", member, "]")
	if s.NextIndex[member] > 1 {
		s.NextIndex[member]--
	}
}
