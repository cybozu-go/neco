package sabakan

import (
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	corev1 "k8s.io/api/core/v1"
)

var (
	errNotAvailable       = errors.New("no healthy machine is available")
	errMissingMachine     = errors.New("failed to apply new template due to missing machines")
	errTooManyNonExistent = errors.New("too many non-existent control plane nodes")

	// DefaultWaitRetiredSeconds before removing retired nodes from the cluster.
	DefaultWaitRetiredSeconds = 300.0
)

// MachineToNode converts sabakan.Machine to cke.Node.
// Add taints, labels, and annotations according to the rules:
//  - https://github.com/cybozu-go/cke/blob/master/docs/sabakan-integration.md#taint-nodes
//  - https://github.com/cybozu-go/cke/blob/master/docs/sabakan-integration.md#node-labels
//  - https://github.com/cybozu-go/cke/blob/master/docs/sabakan-integration.md#node-annotations
func MachineToNode(m *Machine, tmpl *cke.Node) *cke.Node {
	n := &cke.Node{
		Address:      m.Spec.IPv4[0],
		User:         tmpl.User,
		ControlPlane: tmpl.ControlPlane,
		Annotations:  make(map[string]string),
		Labels:       make(map[string]string),
	}

	for k, v := range tmpl.Annotations {
		n.Annotations[k] = v
	}
	n.Annotations["cke.cybozu.com/serial"] = m.Spec.Serial
	n.Annotations["cke.cybozu.com/register-date"] = m.Spec.RegisterDate.Format(time.RFC3339)
	n.Annotations["cke.cybozu.com/retire-date"] = m.Spec.RetireDate.Format(time.RFC3339)

	for _, label := range m.Spec.Labels {
		n.Labels["sabakan.cke.cybozu.com/"+label.Name] = label.Value
	}
	for k, v := range tmpl.Labels {
		n.Labels[k] = v
	}
	n.Labels["cke.cybozu.com/rack"] = strconv.Itoa(m.Spec.Rack)
	n.Labels["cke.cybozu.com/index-in-rack"] = strconv.Itoa(m.Spec.IndexInRack)
	n.Labels["cke.cybozu.com/role"] = m.Spec.Role
	n.Labels["node-role.kubernetes.io/"+m.Spec.Role] = "true"
	if n.ControlPlane {
		n.Labels["node-role.kubernetes.io/master"] = "true"
	}
	n.Labels["topology.kubernetes.io/zone"] = "rack" + strconv.Itoa(m.Spec.Rack)
	n.Labels["failure-domain.beta.kubernetes.io/zone"] = "rack" + strconv.Itoa(m.Spec.Rack)

	for _, taint := range tmpl.Taints {
		n.Taints = append(n.Taints, taint)
	}
	switch m.Status.State {
	case StateUnhealthy:
		n.Taints = append(n.Taints, corev1.Taint{
			Key:    "cke.cybozu.com/state",
			Value:  "unhealthy",
			Effect: corev1.TaintEffectNoSchedule,
		})
	case StateUnreachable:
		n.Taints = append(n.Taints, corev1.Taint{
			Key:    "cke.cybozu.com/state",
			Value:  "unreachable",
			Effect: corev1.TaintEffectNoSchedule,
		})
	case StateRetiring:
		n.Taints = append(n.Taints, corev1.Taint{
			Key:    "cke.cybozu.com/state",
			Value:  "retiring",
			Effect: corev1.TaintEffectNoExecute,
		})
	case StateRetired:
		n.Taints = append(n.Taints, corev1.Taint{
			Key:    "cke.cybozu.com/state",
			Value:  "retired",
			Effect: corev1.TaintEffectNoExecute,
		})
	}

	return n
}

type nodeTemplate struct {
	*cke.Node
	Role   string
	Weight float64
}

// Generator generates cluster configuration.
type Generator struct {
	template    *cke.Cluster
	constraints *cke.Constraints
	timestamp   time.Time
	waitSeconds float64

	machineMap  map[string]*Machine
	cpTmpl      nodeTemplate
	workerTmpls []nodeTemplate

	// intermediate data
	nextUnused        []*Machine
	nextControlPlanes []*Machine
	nextWorkers       []*Machine
	countWorkerByRole map[string]int
}

// NewGenerator creates a new Generator.
// template must have been validated with ValidateTemplate().
func NewGenerator(template *cke.Cluster, cstr *cke.Constraints, machines []Machine, currentTime time.Time) *Generator {
	g := &Generator{
		template:          template,
		constraints:       cstr,
		timestamp:         currentTime,
		waitSeconds:       DefaultWaitRetiredSeconds,
		machineMap:        make(map[string]*Machine),
		countWorkerByRole: make(map[string]int),
	}

	for _, m := range machines {
		if len(m.Spec.IPv4) == 0 {
			log.Warn("ignore machine w/o IPv4 address", map[string]interface{}{
				"serial": m.Spec.Serial,
			})
			continue
		}
		m := m
		g.machineMap[m.Spec.IPv4[0]] = &m
	}

	for _, n := range template.Nodes {
		weight := 1.0
		if val, ok := n.Labels[CKELabelWeight]; ok {
			weight, _ = strconv.ParseFloat(val, 64)
		}
		tmpl := nodeTemplate{
			Node:   n,
			Role:   n.Labels[CKELabelRole],
			Weight: weight,
		}

		if n.ControlPlane {
			g.cpTmpl = tmpl
		} else {
			g.workerTmpls = append(g.workerTmpls, tmpl)
		}
	}

	return g
}

func (g *Generator) chooseWorkerTmpl() nodeTemplate {
	count := g.countWorkerByRole

	least := float64(count[g.workerTmpls[0].Role]) / g.workerTmpls[0].Weight
	leastIndex := 0
	for i, tmpl := range g.workerTmpls {
		w := float64(count[tmpl.Role]) / tmpl.Weight
		if w < least {
			least = w
			leastIndex = i
		}
	}

	return g.workerTmpls[leastIndex]
}

func (g *Generator) getWorkerTmpl(role string) nodeTemplate {
	if len(g.workerTmpls) == 1 {
		return g.workerTmpls[0]
	}

	for _, tmpl := range g.workerTmpls {
		if tmpl.Role == role {
			return tmpl
		}
	}
	panic("BUG: instantiating for invalid role: " + role)
}

// SetWaitSeconds set seconds before removing retired nodes from the cluster.
func (g *Generator) SetWaitSeconds(secs float64) {
	g.waitSeconds = secs
}

// removeMachine removes the specified machine from machines slice.
//  If m is nil or if m is not found in machines, this causes panic.
func removeMachine(machines []*Machine, m *Machine) []*Machine {
	for i, mm := range machines {
		if m == mm {
			return append(machines[:i], machines[i+1:]...)
		}
	}
	panic("BUG: illegal machines operation")
}

// selectWorker selects a healthy machine from given machines slice.
// If there is no such machine, this returns nil.
func (g *Generator) selectWorker(machines []*Machine) *Machine {
	workerTmpl := g.chooseWorkerTmpl()
	candidates := filterHealthyMachinesByRole(machines, workerTmpl.Role)
	if len(candidates) == 0 {
		return nil
	}

	countByRack := g.countMachinesByRack(false, workerTmpl.Role)
	sort.Slice(candidates, func(i, j int) bool {
		si := scoreMachine(candidates[i], countByRack, g.timestamp)
		sj := scoreMachine(candidates[j], countByRack, g.timestamp)
		// higher first
		return si > sj
	})
	return candidates[0]
}

// selectControlPlane selects a healthy controle plane from given machines slice.
// If there is no such machine, this returns nil.
func (g *Generator) selectControlPlane(machines []*Machine) *Machine {
	candidates := filterHealthyMachinesByRole(machines, g.cpTmpl.Role)
	if len(candidates) == 0 {
		return nil
	}

	countByRack := g.countMachinesByRack(true, "")
	sort.Slice(candidates, func(i, j int) bool {
		si := scoreMachine(candidates[i], countByRack, g.timestamp)
		sj := scoreMachine(candidates[j], countByRack, g.timestamp)
		// higher first
		return si > sj
	})
	return candidates[0]
}

// deselectControlPlane selects the lowest scored control plane.
func (g *Generator) deselectControlPlane() *Machine {
	countByRack := g.countMachinesByRack(true, "")
	sort.Slice(g.nextControlPlanes, func(i, j int) bool {
		si := scoreMachineWithHealthStatus(g.nextControlPlanes[i], countByRack, g.timestamp)
		sj := scoreMachineWithHealthStatus(g.nextControlPlanes[j], countByRack, g.timestamp)
		// lower first
		return si < sj
	})
	return g.nextControlPlanes[0]
}

// fill allocates new machines and/or promotes excessive workers to control plane
// to satisfy given constraints, then generate cluster configuration.
func (g *Generator) fill(op *updateOp) (*cke.Cluster, error) {
	for i := len(g.nextControlPlanes); i < g.constraints.ControlPlaneCount; i++ {
		m := g.selectControlPlane(g.nextUnused)
		if m != nil {
			op.addControlPlane(m)
			g.nextControlPlanes = append(g.nextControlPlanes, m)
			g.nextUnused = removeMachine(g.nextUnused, m)
			continue
		}

		// If no unused machines available, steal a redundant worker and promote it as a control plane.
		if len(g.nextWorkers) > g.constraints.MinimumWorkers {
			promote := g.selectControlPlane(g.nextWorkers)
			if promote != nil {
				op.promoteWorker(promote)
				g.nextControlPlanes = append(g.nextControlPlanes, promote)
				g.removeNextWorker(promote)
				continue
			}
		}
		return nil, errNotAvailable
	}

	for i := len(g.nextWorkers); i < g.constraints.MinimumWorkers; i++ {
		m := g.selectWorker(g.nextUnused)
		if m == nil {
			return nil, errNotAvailable
		}
		op.addWorker(m)
		g.appendNextWorker(m)
		g.nextUnused = removeMachine(g.nextUnused, m)
	}

	err := log.Info("sabakan: generated cluster", map[string]interface{}{
		"op":      op.name,
		"changes": op.changes,
	})
	if err != nil {
		panic(err)
	}

	nodes := make([]*cke.Node, 0, len(g.nextControlPlanes)+len(g.nextWorkers))
	for _, m := range g.nextControlPlanes {
		nodes = append(nodes, MachineToNode(m, g.cpTmpl.Node))
	}
	for _, m := range g.nextWorkers {
		nodes = append(nodes, MachineToNode(m, g.getWorkerTmpl(m.Spec.Role).Node))
	}

	c := *g.template
	c.Nodes = nodes
	return &c, nil
}

// Generate generates a new *Cluster that satisfies constraints.
func (g *Generator) Generate() (*cke.Cluster, error) {
	op := &updateOp{
		name: "new",
	}
	op.record("generate new cluster")

	g.nextUnused = g.getUnusedMachines(nil)

	return g.fill(op)
}

// Regenerate regenerates *Cluster using the same set of nodes in the current configuration.
// This method should be used only when the template is updated and no other changes happen.
func (g *Generator) Regenerate(current *cke.Cluster) (*cke.Cluster, error) {
	op := &updateOp{
		name: "regenerate",
	}

	nextCPs, nonExistentCPs := g.nodesToMachines(cke.ControlPlanes(current.Nodes))
	if nonExistentCPs != nil {
		return nil, errMissingMachine
	}
	g.nextControlPlanes = nextCPs

	nextWorkers, nonExistentWorkers := g.nodesToMachines(cke.Workers(current.Nodes))
	if nonExistentWorkers != nil {
		return nil, errMissingMachine
	}

	for _, m := range nextWorkers {
		g.appendNextWorker(m)
	}
	g.nextUnused = g.getUnusedMachines(current.Nodes)

	op.record("regenerate with new template")
	return g.fill(op)
}

// Update updates the current configuration when necessary.
// If the generator decides no updates are necessary, it returns (nil, nil).
func (g *Generator) Update(current *cke.Cluster) (*cke.Cluster, error) {
	currentCPs := cke.ControlPlanes(current.Nodes)
	currentWorkers := cke.Workers(current.Nodes)

	op, err := g.removeNonExistentNode(currentCPs, currentWorkers)
	if err != nil {
		return nil, err
	}
	if op != nil {
		return g.fill(op)
	}

	op, err = g.increaseControlPlane()
	if err != nil {
		return nil, err
	}
	if op != nil {
		return g.fill(op)
	}

	op, err = g.decreaseControlPlane()
	if err != nil {
		return nil, err
	}
	if op != nil {
		return g.fill(op)
	}

	op, err = g.replaceControlPlane()
	if err != nil {
		return nil, err
	}
	if op != nil {
		return g.fill(op)
	}

	op, err = g.increaseWorker()
	if err != nil {
		return nil, err
	}
	if op != nil {
		return g.fill(op)
	}

	op, err = g.decreaseWorker()
	if err != nil {
		return nil, err
	}
	if op != nil {
		return g.fill(op)
	}

	op, err = g.taintNodes(currentCPs, currentWorkers)
	if err != nil {
		return nil, err
	}
	if op != nil {
		return g.fill(op)
	}

	return nil, nil
}

func (g *Generator) removeNonExistentNode(currentCPs, currentWorkers []*cke.Node) (*updateOp, error) {
	op := &updateOp{
		name: "remove non-existent node",
	}

	nextCPs, nonExistentCPs := g.nodesToMachines(currentCPs)
	if len(nextCPs)*2 <= len(currentCPs) {
		// Replacing more than half of control plane nodes would destroy
		// etcd cluster.  We cannot do anything in this case.
		return nil, errTooManyNonExistent
	}
	for _, n := range nonExistentCPs {
		op.record("remove non-existent control plane: " + n.Address)
	}
	g.nextControlPlanes = nextCPs

	nextWorkers, nonExistentWorkers := g.nodesToMachines(currentWorkers)
	for _, n := range nonExistentWorkers {
		op.record("remove non-existent worker: " + n.Address)
	}
	for _, m := range nextWorkers {
		g.appendNextWorker(m)
	}

	g.nextUnused = g.getUnusedMachines(append(currentCPs, currentWorkers...))

	if len(op.changes) == 0 {
		// nothing to do
		return nil, nil
	}

	return op, nil
}

func (g *Generator) increaseControlPlane() (*updateOp, error) {
	if len(g.nextControlPlanes) >= g.constraints.ControlPlaneCount {
		return nil, nil
	}

	op := &updateOp{
		name: "increase control plane",
	}

	return op, nil
}

func (g *Generator) decreaseControlPlane() (*updateOp, error) {
	if len(g.nextControlPlanes) <= g.constraints.ControlPlaneCount {
		return nil, nil
	}

	op := &updateOp{
		name: "decrease control plane",
	}

	for len(g.nextControlPlanes) > g.constraints.ControlPlaneCount {
		m := g.deselectControlPlane()
		g.nextControlPlanes = removeMachine(g.nextControlPlanes, m)

		if g.constraints.MaximumWorkers == 0 || len(g.nextWorkers) < g.constraints.MaximumWorkers {
			op.demoteControlPlane(m)
			g.appendNextWorker(m)
			continue
		}
		op.record("remove excessive control plane: " + m.Spec.IPv4[0])
		g.nextUnused = append(g.nextUnused, m)
	}

	return op, nil
}

func (g *Generator) replaceControlPlane() (*updateOp, error) {
	// If there is only one control plane, this algorithm cannot be chosen.
	if len(g.nextControlPlanes) < 2 {
		return nil, nil
	}

	var demote *Machine
	for _, m := range g.nextControlPlanes {
		state := m.Status.State
		if !(state == StateHealthy || state == StateUpdating || state == StateUninitialized) {
			demote = m
			break
		}
	}
	if demote == nil {
		return nil, nil
	}

	op := &updateOp{
		name: "replace control plane",
	}
	g.nextControlPlanes = removeMachine(g.nextControlPlanes, demote)

	if g.constraints.MaximumWorkers == 0 || len(g.nextWorkers) < g.constraints.MaximumWorkers {
		op.demoteControlPlane(demote)
		g.appendNextWorker(demote)
		return op, nil
	}

	promote := g.selectControlPlane(g.nextWorkers)
	if promote == nil {
		op.record("remove bad control plane: " + demote.Spec.IPv4[0])
		return op, nil
	}

	op.promoteWorker(promote)
	g.nextControlPlanes = append(g.nextControlPlanes, promote)
	g.removeNextWorker(promote)
	g.appendNextWorker(demote)

	return op, nil
}

func (g *Generator) increaseWorker() (*updateOp, error) {
	var healthyWorkers int
	for _, m := range g.nextWorkers {
		if m.Status.State == StateHealthy {
			healthyWorkers++
		}
	}

	if healthyWorkers >= g.constraints.MinimumWorkers {
		return nil, nil
	}

	op := &updateOp{
		name: "increase worker",
	}

	for i := healthyWorkers; i < g.constraints.MinimumWorkers; i++ {
		if g.constraints.MaximumWorkers != 0 && len(g.nextWorkers) >= g.constraints.MaximumWorkers {
			break
		}
		m := g.selectWorker(g.nextUnused)
		if m == nil {
			break
		}
		op.addWorker(m)
		g.appendNextWorker(m)
		g.nextUnused = removeMachine(g.nextUnused, m)
	}

	if len(op.changes) == 0 {
		return nil, nil
	}

	return op, nil
}

func (g *Generator) decreaseWorker() (*updateOp, error) {
	var retired *Machine
	for _, m := range g.nextWorkers {
		if m.Status.State == StateRetired && m.Status.Duration > g.waitSeconds {
			retired = m
			break
		}
	}
	if retired == nil {
		return nil, nil
	}

	op := &updateOp{
		name: "decrease worker",
	}

	healthyUnused := g.selectWorker(g.nextUnused)
	if healthyUnused == nil && len(g.nextWorkers)-1 < g.constraints.MinimumWorkers {
		// in this condition, CKE cannot decrease worker because `minimum-workers` will not be filled.
		return nil, nil
	}

	op.record("remove retired worker: " + retired.Spec.IPv4[0])
	g.removeNextWorker(retired)
	return op, nil
}

func (g *Generator) nodesToMachines(nodes []*cke.Node) ([]*Machine, []*cke.Node) {
	var machines []*Machine
	var nonExistent []*cke.Node
	for _, n := range nodes {
		if m := g.machineMap[n.Address]; m == nil {
			nonExistent = append(nonExistent, n)
			continue
		}
		machines = append(machines, g.machineMap[n.Address])
	}
	return machines, nonExistent
}

func (g *Generator) getUnusedMachines(usedNodes []*cke.Node) []*Machine {
	nodeMap := make(map[string]*cke.Node)
	for _, n := range usedNodes {
		nodeMap[n.Address] = n
	}

	var unused []*Machine
	for a, m := range g.machineMap {
		if _, ok := nodeMap[a]; !ok && m.Status.State == StateHealthy {
			unused = append(unused, m)
		}
	}

	return unused
}

func hasValidTaint(n *cke.Node, m *Machine) bool {
	var ckeTaint corev1.Taint
	for _, taint := range n.Taints {
		if taint.Key == "cke.cybozu.com/state" {
			ckeTaint = taint
			break
		}
	}

	switch m.Status.State {
	case StateUnhealthy:
		if ckeTaint.Value != "unhealthy" || ckeTaint.Effect != corev1.TaintEffectNoSchedule {
			return false
		}
	case StateUnreachable:
		if ckeTaint.Value != "unreachable" || ckeTaint.Effect != corev1.TaintEffectNoSchedule {
			return false
		}
	case StateRetiring:
		if ckeTaint.Value != "retiring" || ckeTaint.Effect != corev1.TaintEffectNoExecute {
			return false
		}
	case StateRetired:
		if ckeTaint.Value != "retired" || ckeTaint.Effect != corev1.TaintEffectNoExecute {
			return false
		}
	default:
		if ckeTaint.Key != "" {
			return false
		}
	}

	return true
}

func (g *Generator) taintNodes(currentCPs, currentWorkers []*cke.Node) (*updateOp, error) {
	op := &updateOp{
		name: "taint nodes",
	}

	for _, n := range currentCPs {
		m := g.machineMap[n.Address]
		if m == nil {
			panic("BUG: " + n.Address + " does not exist")
		}
		if !hasValidTaint(n, m) {
			op.record("change taint of " + n.Address)
		}
	}

	for _, n := range currentWorkers {
		m := g.machineMap[n.Address]
		if m == nil {
			panic("BUG: " + n.Address + " does not exist")
		}
		if !hasValidTaint(n, m) {
			op.record("change taint of " + n.Address)
		}
	}

	if len(op.changes) == 0 {
		return nil, nil
	}

	return op, nil
}

func (g *Generator) countMachinesByRack(cp bool, role string) map[int]int {
	machines := g.nextControlPlanes
	if !cp {
		machines = g.nextWorkers
	}

	count := make(map[int]int)
	for _, m := range machines {
		if !cp && role != "" && role != m.Spec.Role {
			continue
		}
		count[m.Spec.Rack]++
	}
	return count
}

func (g *Generator) appendNextWorker(m *Machine) {
	g.nextWorkers = append(g.nextWorkers, m)
	g.countWorkerByRole[m.Spec.Role]++
}

func (g *Generator) removeNextWorker(m *Machine) {
	g.nextWorkers = removeMachine(g.nextWorkers, m)
	g.countWorkerByRole[m.Spec.Role]--
}
