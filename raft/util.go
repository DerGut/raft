package raft

import "github.com/DerGut/raft/raft/state"

func isBehind(reqTerm, currentTerm state.Term) bool { return reqTerm < currentTerm }
