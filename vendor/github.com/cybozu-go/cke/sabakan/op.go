package sabakan

type updateOp struct {
	name    string
	changes []string
	cps     []*Machine
	workers []*Machine
}

func (op *updateOp) record(msg string) {
	op.changes = append(op.changes, msg)
}

func (op *updateOp) addControlPlane(m *Machine) {
	op.record("add new control plane: " + m.Spec.IPv4[0])
	op.cps = append(op.cps, m)
}

func (op *updateOp) addWorker(m *Machine) {
	op.record("add new worker: " + m.Spec.IPv4[0])
	op.workers = append(op.workers, m)
}

func (op *updateOp) promoteWorker() bool {
	var cp *Machine
	for i, m := range op.workers {
		if m.Status.State == StateHealthy {
			cp = m
			op.workers = append(op.workers[:i], op.workers[i+1:]...)
			break
		}
	}
	if cp == nil {
		return false
	}

	op.record("promote a worker: " + cp.Spec.IPv4[0])
	op.cps = append(op.cps, cp)
	return true
}

func (op *updateOp) demoteControlPlane(cp *Machine) {
	op.record("demote a control plane: " + cp.Spec.IPv4[0])
	op.workers = append(op.workers, cp)
}
