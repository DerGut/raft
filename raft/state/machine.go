package state

// Machine represents the state machine running on top of the raft protocol.
// The user can define this itself. If the raft cluster has agreed on a new value,
// it will be commited to the state machine.
type Machine interface {
	Commit(cmds []string)
}
