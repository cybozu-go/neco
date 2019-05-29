//go:generate go run ./scripts/gqlgen.go

package gql

import (
	"context"
	"net"
	"sort"
	"time"

	"github.com/cybozu-go/sabakan/v2"
)

// Resolver implements ResolverRoot.
type Resolver struct {
	Model sabakan.Model
}

// BMC implements ResolverRoot.
func (r *Resolver) BMC() BMCResolver {
	return &bMCResolver{r}
}

// NICConfig implements ResolverRoot.
func (r *Resolver) NICConfig() NICConfigResolver {
	return &nICConfigResolver{r}
}

// MachineSpec implements ResolverRoot.
func (r *Resolver) MachineSpec() MachineSpecResolver {
	return &machineSpecResolver{r}
}

// MachineStatus implements ResolverRoot.
func (r *Resolver) MachineStatus() MachineStatusResolver {
	return &machineStatusResolver{r}
}

// Query implements ResolverRoot.
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}

type bMCResolver struct{ *Resolver }

func (r *bMCResolver) BmcType(ctx context.Context, obj *sabakan.MachineBMC) (string, error) {
	return obj.Type, nil
}
func (r *bMCResolver) Ipv4(ctx context.Context, obj *sabakan.MachineBMC) (IPAddress, error) {
	return IPAddress(net.ParseIP(obj.IPv4)), nil
}

type nICConfigResolver struct{ *Resolver }

func (r *nICConfigResolver) Address(ctx context.Context, obj *sabakan.NICConfig) (IPAddress, error) {
	return IPAddress(net.ParseIP(obj.Address)), nil
}
func (r *nICConfigResolver) Netmask(ctx context.Context, obj *sabakan.NICConfig) (IPAddress, error) {
	return IPAddress(net.ParseIP(obj.Netmask)), nil
}
func (r *nICConfigResolver) Gateway(ctx context.Context, obj *sabakan.NICConfig) (IPAddress, error) {
	return IPAddress(net.ParseIP(obj.Gateway)), nil
}

type machineSpecResolver struct{ *Resolver }

func (r *machineSpecResolver) Labels(ctx context.Context, obj *sabakan.MachineSpec) ([]Label, error) {
	if len(obj.Labels) == 0 {
		return []Label{}, nil
	}

	keys := make([]string, 0, len(obj.Labels))
	for k := range obj.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	labels := make([]Label, 0, len(obj.Labels))
	for _, k := range keys {
		labels = append(labels, Label{Name: k, Value: obj.Labels[k]})
	}
	return labels, nil
}
func (r *machineSpecResolver) Rack(ctx context.Context, obj *sabakan.MachineSpec) (int, error) {
	return int(obj.Rack), nil
}
func (r *machineSpecResolver) IndexInRack(ctx context.Context, obj *sabakan.MachineSpec) (int, error) {
	return int(obj.IndexInRack), nil
}
func (r *machineSpecResolver) Ipv4(ctx context.Context, obj *sabakan.MachineSpec) ([]IPAddress, error) {
	addresses := make([]IPAddress, len(obj.IPv4))
	for i, a := range obj.IPv4 {
		addresses[i] = IPAddress(net.ParseIP(a))
	}
	return addresses, nil
}
func (r *machineSpecResolver) RegisterDate(ctx context.Context, obj *sabakan.MachineSpec) (DateTime, error) {
	return DateTime(obj.RegisterDate), nil
}
func (r *machineSpecResolver) RetireDate(ctx context.Context, obj *sabakan.MachineSpec) (DateTime, error) {
	return DateTime(obj.RetireDate), nil
}

type machineStatusResolver struct{ *Resolver }

func (r *machineStatusResolver) Timestamp(ctx context.Context, obj *sabakan.MachineStatus) (DateTime, error) {
	return DateTime(obj.Timestamp), nil
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Machine(ctx context.Context, serial string) (sabakan.Machine, error) {
	now := time.Now()
	machine, err := r.Model.Machine.Get(ctx, serial)
	if err != nil {
		return sabakan.Machine{}, err
	}
	machine.Status.Duration = now.Sub(machine.Status.Timestamp).Seconds()
	return *machine, nil
}
func (r *queryResolver) SearchMachines(ctx context.Context, having, notHaving *MachineParams) ([]sabakan.Machine, error) {
	now := time.Now()
	machines, err := r.Model.Machine.Query(ctx, sabakan.Query{})
	if err != nil {
		return nil, err
	}
	var filtered []sabakan.Machine
	for _, m := range machines {
		m.Status.Duration = now.Sub(m.Status.Timestamp).Seconds()
		if matchMachine(m, having, notHaving, now) {
			filtered = append(filtered, *m)
		}
	}
	return filtered, nil
}
