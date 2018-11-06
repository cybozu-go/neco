package worker

// Barrier is used for barrier synchronization between boot servers.
type Barrier map[int]struct{}

// NewBarrier returns Barrier to synchronize boot servers listed in lrns.
func NewBarrier(lrns []int) Barrier {
	m := make(map[int]struct{})
	for _, lrn := range lrns {
		m[lrn] = struct{}{}
	}
	return Barrier(m)
}

// Check inform the barrier of the arrival of a boot server to a check point.
// It returns true when all boot servers reache the check point.
func (b Barrier) Check(lrn int) bool {
	delete(b, lrn)
	return len(b) == 0
}
