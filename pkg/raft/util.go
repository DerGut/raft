package raft

import "github.com/DerGut/raft/pkg/state"

func isBehind(reqTerm, currentTerm state.Term) bool { return reqTerm < currentTerm }
