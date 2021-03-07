package app

import "log"

type StateMachine struct {
	entries []string
}

func (m *StateMachine) Commit(cmds []string) {
	log.Println("Committed:", cmds)
	m.entries = append(m.entries, cmds...)
	log.Println("Now:", m.entries)
}
