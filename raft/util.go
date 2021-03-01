package raft

import "github.com/DerGut/kv-store/raft/state"

func isBehind(reqTerm, currentTerm state.Term) bool { return reqTerm < currentTerm }
