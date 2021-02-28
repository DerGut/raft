package raft

import (
	"github.com/DerGut/kv-store/replog"
)

func isBehind(reqTerm, currentTerm replog.Term) bool { return reqTerm < currentTerm }
