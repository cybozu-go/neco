package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

type updateBlockPVsUpToV1_16Op struct {
	apiServer *cke.Node
	nodes     []*cke.Node
	pvs       map[string][]string
	step      int
}

// UpdateBlockPVsUpToV1_16Op returns an Operator to restart kubelet
func UpdateBlockPVsUpToV1_16Op(apiServer *cke.Node, nodes []*cke.Node, pvs map[string][]string) cke.Operator {
	return &updateBlockPVsUpToV1_16Op{apiServer: apiServer, nodes: nodes, pvs: pvs}
}

func (o *updateBlockPVsUpToV1_16Op) Name() string {
	return "update-block-up-to-1.16"
}

func (o *updateBlockPVsUpToV1_16Op) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}

func (o *updateBlockPVsUpToV1_16Op) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return updateBlockPVsUpTo1_16(o.apiServer, o.nodes, o.pvs)
	default:
		return nil
	}
}

type updateBlockPVsUpTo1_16Command struct {
	apiServer *cke.Node
	nodes     []*cke.Node
	pvs       map[string][]string
}

// updateBlockPVsUpTo1_16 move raw block device files.
// This command is used for upgrading to k8s 1.17
func updateBlockPVsUpTo1_16(apiServer *cke.Node, nodes []*cke.Node, pvs map[string][]string) cke.Commander {
	return updateBlockPVsUpTo1_16Command{apiServer: apiServer, nodes: nodes, pvs: pvs}
}

func (c updateBlockPVsUpTo1_16Command) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	type ckeToolResult struct {
		Result string `json:"result"`
	}

	begin := time.Now()
	env := well.NewEnvironment(ctx)
	for _, node := range c.nodes {
		node := node
		pvs := c.pvs[node.Address]

		env.Go(func(ctx context.Context) error {
			ce := inf.Engine(node.Address)

			for _, pv := range pvs {
				arg := strings.Join([]string{
					"/usr/local/cke-tools/bin/updateblock117",
					"operate",
					pv,
				}, " ")
				binds := []cke.Mount{
					{
						Source:      "/var/lib/kubelet",
						Destination: "/var/lib/kubelet",
						Label:       cke.LabelPrivate,
					},
				}
				stdout, stderr, err := ce.RunWithOutput(cke.ToolsImage, binds, arg)
				if err != nil || len(stderr) != 0 {
					return fmt.Errorf("updateblock117 operate failed, %w, stdout: %s, stderr: %s", err, string(stdout), string(stderr))
				}
				// parse stdout
				var result ckeToolResult
				err = json.Unmarshal(stdout, &result)
				if err != nil {
					return fmt.Errorf("unmarshal error, %w, stdout: %s", err, string(stdout))
				}
				if result.Result != "completed" {
					return fmt.Errorf("updateblock117 operate result failed, stdout: %s", string(stdout))
				}
			}
			return nil
		})
	}
	env.Stop()
	err := env.Wait()
	log.Info("updateBlockPVsUpTo1_16 finished", map[string]interface{}{
		"elapsed": time.Now().Sub(begin).Seconds(),
	})
	return err
}

func (c updateBlockPVsUpTo1_16Command) Command() cke.Command {
	return cke.Command{Name: "update-block-up-to-1.16"}
}
