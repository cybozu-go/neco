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

func (op *updateOp) promoteWorker(worker *Machine) bool {
	if len(worker.Spec.IPv4) == 0 {
		panic("the given worker's IP address is missing")
	}

	var cp *Machine
	for i, m := range op.workers {
		if len(m.Spec.IPv4) == 0 {
			continue
		}
		if m.Spec.IPv4[0] == worker.Spec.IPv4[0] {
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

func (op *updateOp) countMachinesByRack(isCP bool) map[int]int {
	machines := op.cps
	if !isCP {
		machines = op.workers
	}

	count := make(map[int]int)
	for _, m := range machines {
		count[m.Spec.Rack]++
	}
	return count
}
