package sabakan

type updateOp struct {
	name    string
	changes []string
}

func (op *updateOp) record(msg string) {
	op.changes = append(op.changes, msg)
}

func (op *updateOp) addControlPlane(m *Machine) {
	op.record("add new control plane: " + m.Spec.IPv4[0])
}

func (op *updateOp) addWorker(m *Machine) {
	op.record("add new worker: " + m.Spec.IPv4[0])
}

func (op *updateOp) promoteWorker(worker *Machine) {
	op.record("promote a worker: " + worker.Spec.IPv4[0])
}

func (op *updateOp) demoteControlPlane(cp *Machine) {
	op.record("demote a control plane: " + cp.Spec.IPv4[0])
}
