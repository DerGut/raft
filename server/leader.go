package server

import "github.com/DerGut/kv-store/replog"

func (s *Server) sendHeartBeat() {
	req := s.buildAppendEntriesRequest([]replog.Entry{})

	resC := make(chan *AppendEntriesResponse, len(s.Cluster))
	errC := make(chan error, len(s.Cluster))
	s.Cluster.callAppendEntriesOnAll(&req, resC, errC)

	for i := 0; i < len(s.Cluster); i++ {
		select {
		case res := <-resC:
			if isBehind(s.state, res.Term) {
				s.updateTerm(res.Term)
				s.returnToFollower()
				return
			}
		case <-errC:
			continue
		}
	}
}

func (s *Server) buildAppendEntriesRequest(entries []replog.Entry) AppendEntriesRequest {
	l := s.state.Log()
	return AppendEntriesRequest{
		Term:         s.state.CurrentTerm(),
		LeaderID:     s.MemberID,
		PrevLogIndex: l.LastIndex(),
		PrevLogTerm:  l.LastTerm(),
		Entries:      entries,
		LeaderCommit: s.state.CommitIndex(),
	}
}
