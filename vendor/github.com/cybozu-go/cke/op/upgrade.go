package op

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
)

type upgradeOp struct {
	current string
	nodes   []*cke.Node
}

// UpgradeOp returns an Operator to upgrade cluster configuration.
func UpgradeOp(current string, nodes []*cke.Node) cke.Operator {
	return &upgradeOp{current: current, nodes: nodes}
}

func (u *upgradeOp) Name() string {
	return "upgrade"
}

func (u *upgradeOp) NextCommand() cke.Commander {
	switch u.current {
	case "1":
		u.current = "2"
		return UpgradeToVersion2Command(u.nodes)
	default:
		return nil
	}
}

func (u *upgradeOp) Targets() []string {
	targets := make([]string, len(u.nodes))
	for i, n := range u.nodes {
		targets[i] = n.Address
	}
	return targets
}

type upgradeToVersion2Command struct {
	nodes []*cke.Node
}

// UpgradeToVersion2Command returns a Commander to upgrade from version 1 to 2.
func UpgradeToVersion2Command(nodes []*cke.Node) cke.Commander {
	return upgradeToVersion2Command{nodes}
}

func (u upgradeToVersion2Command) Run(ctx context.Context, inf cke.Infrastructure, leaderKey string) error {
	env := well.NewEnvironment(ctx)
	for _, n := range u.nodes {
		ce := inf.Engine(n.Address)
		env.Go(func(ctx context.Context) error {
			exists, err := ce.VolumeExists(EtcdAddedMemberVolumeName)
			if err != nil {
				return err
			}
			if !exists {
				return ce.VolumeCreate(EtcdAddedMemberVolumeName)
			}
			return nil
		})
	}
	env.Stop()
	if err := env.Wait(); err != nil {
		return err
	}

	return inf.Storage().PutConfigVersion(ctx, leaderKey)
}

func (u upgradeToVersion2Command) Command() cke.Command {
	return cke.Command{
		Name:   "upgrade-version-from-1-to-2",
		Target: EtcdAddedMemberVolumeName,
	}
}
